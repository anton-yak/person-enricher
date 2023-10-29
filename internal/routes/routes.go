package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/anton-yak/person-enricher/internal/model"
	postgres "github.com/anton-yak/person-enricher/internal/repository"
)

type ctxKey string

type Logger interface {
	Infof(template string, args ...interface{})
	Debugf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
}

func MakeHTTPHandler(enricher model.Enricher, pgxPool *pgxpool.Pool, logger Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, ctxKey("enricher"), enricher)
			ctx = context.WithValue(ctx, ctxKey("logger"), logger)

			tx, err := pgxPool.Begin(ctx)
			if err != nil {
				logger.Errorf("failed to open transaction: %v", err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}

			postgresRepository := postgres.NewPostgresRepository(tx, logger)
			ctx = context.WithValue(ctx, ctxKey("repository"), &postgresRepository)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Route("/persons", func(r chi.Router) {
		r.Get("/", getAllPersons)
		r.Post("/", createOrUpdatePerson)

		r.Route("/{personID}", func(r chi.Router) {
			r.Use(PersonCtx)
			r.Put("/", createOrUpdatePerson)
			r.Delete("/", deletePerson)
		})
	})

	return r
}

func PersonCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		personID := chi.URLParam(r, "personID")
		pID, err := strconv.ParseUint(personID, 10, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}

		repository := r.Context().Value(ctxKey("repository")).(model.Repository)
		p, err := model.GetPersonWithLock(repository, uint(pID))
		if err != nil {
			repository.Rollback()

			if errors.Is(err, model.ErrPersonNotFound) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			} else {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		ctx := context.WithValue(r.Context(), ctxKey("person"), p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func createOrUpdatePerson(w http.ResponseWriter, r *http.Request) {
	var err error

	ctx := r.Context()

	logger := ctx.Value(ctxKey("logger")).(Logger)

	d := json.NewDecoder(r.Body)
	var person model.Person
	err = d.Decode(&person)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err = person.Validate()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err = person.Enrich(ctx.Value(ctxKey("enricher")).(model.Enricher))
	if err != nil {
		logger.Errorf("failed to enrich person: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// if person already exists we can get ID from value from context which was loaded before
	// and then person.Save() will use UPDATE instead of INSERT
	// maybe it's better to write separate handler for update, but at this point code is almost the same
	if ctx.Value(ctxKey("person")) != nil {
		person.ID = ctx.Value(ctxKey("person")).(*model.Person).ID
		logger.Infof("updating person: %v", person)
	} else {
		logger.Infof("creating person: %v", person)
	}

	repository := ctx.Value(ctxKey("repository")).(model.Repository)
	err = person.Save(repository)
	if err != nil {
		repository.Rollback()
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	repository.Commit()

	w.Header().Add("Content-Type", "application/json")
	e := json.NewEncoder(w)
	err = e.Encode(person)
	if err != nil {
		logger.Errorf("failed to send response in json: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func deletePerson(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := ctx.Value(ctxKey("logger")).(Logger)

	person := ctx.Value(ctxKey("person")).(*model.Person)
	repository := ctx.Value(ctxKey("repository")).(model.Repository)

	logger.Infof("deleting person: %v", person)

	err := person.Delete(repository)
	if err != nil {
		logger.Errorf("failed to delete person: %v", err)
		repository.Rollback()
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	repository.Commit()
	w.Header().Add("Content-Type", "application/json")
	e := json.NewEncoder(w)
	err = e.Encode(person)
	if err != nil {
		logger.Errorf("failed to send response in json: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func getAllPersons(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := ctx.Value(ctxKey("logger")).(Logger)
	repository := ctx.Value(ctxKey("repository")).(model.Repository)

	query := r.URL.Query()

	var id uint
	if query.Has("id") {
		i, err := strconv.ParseUint(query.Get("id"), 10, 64)
		if err != nil {
			logger.Debugf("failed to parse age: %v", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		id = uint(i)
	}
	var age uint
	if query.Has("age") {
		a, err := strconv.ParseUint(query.Get("age"), 10, 64)
		if err != nil {
			logger.Debugf("failed to parse age: %v", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		age = uint(a)
	}
	person := model.Person{
		ID:          id,
		Name:        query.Get("name"),
		Surname:     query.Get("surname"),
		Patronymic:  query.Get("patronymic"),
		Age:         age,
		Gender:      query.Get("gender"),
		Nationality: query.Get("nationality"),
	}
	var limit *uint
	var offset uint
	if query.Has("limit") {
		l64, err := strconv.ParseUint(query.Get("limit"), 10, 64)
		if err != nil {
			logger.Debugf("failed to parse limit: %v", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		l := uint(l64)
		limit = &l
	}
	if query.Has("offset") {
		o, err := strconv.ParseUint(query.Get("offset"), 10, 64)
		if err != nil {
			logger.Debugf("failed to parse offset: %v", err)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		offset = uint(o)
	}
	persons, total, err := model.GetAllPersons(repository, &person, limit, offset)
	if err != nil {
		logger.Errorf("failed to get all persons: %v", err)
		repository.Rollback()
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	repository.Commit()

	w.Header().Add("Content-Type", "application/json")
	e := json.NewEncoder(w)
	err = e.Encode(struct {
		Persons []model.Person `json:"persons"`
		Total   uint           `json:"total"`
	}{
		Persons: persons,
		Total:   total,
	})
	if err != nil {
		logger.Errorf("failed to send response in json: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}
