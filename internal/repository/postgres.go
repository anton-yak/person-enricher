package postgres

import (
	"context"
	"fmt"

	"github.com/anton-yak/person-enricher/internal/model"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

type PostgresRepository struct {
	tx     pgx.Tx
	logger *zap.SugaredLogger
}

func NewPostgresRepository(tx pgx.Tx, logger *zap.SugaredLogger) PostgresRepository {
	return PostgresRepository{
		tx:     tx,
		logger: logger,
	}
}

func (r *PostgresRepository) InsertPerson(p *model.Person) (uint, error) {
	var id uint
	err := r.tx.QueryRow(
		context.Background(),
		`INSERT INTO persons (name, surname, patronymic, age, gender, nationality) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		p.Name,
		p.Surname,
		p.Patronymic,
		p.Age,
		p.Gender,
		p.Nationality,
	).Scan(&id)

	if err != nil {
		r.logger.Errorf("failed to insert person %v", err)
		return 0, err
	}

	return id, nil
}

func (r *PostgresRepository) UpdatePerson(p *model.Person) error {
	cmdTag, err := r.tx.Exec(
		context.Background(),
		`UPDATE persons SET
			name = $1,
			surname = $2,
			patronymic = $3,
			age = $4,
			gender = $5,
			nationality = $6
		WHERE id = $7`,
		p.Name, p.Surname, p.Patronymic, p.Age, p.Gender, p.Nationality, p.ID)

	if err != nil {
		r.logger.Errorf("failed to update person %v", err)
		return err
	}

	if cmdTag.RowsAffected() != 1 {
		err = fmt.Errorf("updated %d rows instead of 1", cmdTag.RowsAffected())
		r.logger.Errorf("failed to update person %v", err)
		return err
	}

	return nil
}

func (r *PostgresRepository) DeletePerson(id uint) error {
	cmdTag, err := r.tx.Exec(context.Background(), `DELETE FROM persons WHERE id = $1`, id)

	if err != nil {
		r.logger.Errorf("failed to delete person %v", err)
		return err
	}

	if cmdTag.RowsAffected() != 1 {
		err = fmt.Errorf("deleted %d rows instead of 1", cmdTag.RowsAffected())
		r.logger.Errorf("failed to delete person %v", err)
		return err
	}

	return nil
}

func (r *PostgresRepository) GetPersonWithLock(id uint) (*model.Person, error) {
	var p model.Person
	err := r.tx.QueryRow(
		context.Background(),
		`SELECT id, name, surname, patronymic, age, gender, nationality
		FROM persons
		WHERE id = $1
		FOR UPDATE`, id).Scan(&p.ID, &p.Name, &p.Surname, &p.Patronymic, &p.Age, &p.Gender, &p.Nationality)

	if err != nil {
		r.logger.Errorf("failed to select person for update %v", err)
		return nil, err
	}

	return &p, nil
}

func (r *PostgresRepository) GetAllPersons(p *model.Person, limit *uint, offset uint) ([]model.Person, uint, error) {
	namedArgs := pgx.NamedArgs{
		"id":          p.ID,
		"name":        p.Name,
		"surname":     p.Surname,
		"patronymic":  p.Patronymic,
		"age":         p.Age,
		"gender":      p.Gender,
		"nationality": p.Nationality,
		"offset":      offset,
	}

	fromWhereSql := `FROM persons
		WHERE (id = @id OR @id = 0)
		  AND (name = @name OR @name = '')
		  AND (surname = @surname OR @surname = '')
		  AND (age = @age OR @age = 0)
		  AND (gender = @gender OR @gender = '')
		  AND (nationality = @nationality OR @nationality = '')`

	var total uint
	err := r.tx.QueryRow(context.Background(), `SELECT count(1) `+fromWhereSql, namedArgs).Scan(&total)
	if err != nil {
		r.logger.Errorf("failed to count persons %v", err)
		return nil, 0, err
	}

	var limitSql string

	if limit != nil {
		limitSql = "LIMIT @limit"
		namedArgs["limit"] = *limit
	}

	rows, err := r.tx.Query(
		context.Background(),
		`SELECT id, name, surname, patronymic, age, gender, nationality `+fromWhereSql+` `+limitSql+` OFFSET @offset`,
		namedArgs,
	)
	if err != nil {
		r.logger.Errorf("failed to select persons %v", err)
		return nil, 0, err
	}
	defer rows.Close()

	persons := []model.Person{}
	for rows.Next() {
		var p model.Person
		err := rows.Scan(&p.ID, &p.Name, &p.Surname, &p.Patronymic, &p.Age, &p.Gender, &p.Nationality)
		if err != nil {
			r.logger.Errorf("failed to scan row %v", err)
			return nil, 0, err
		}
		persons = append(persons, p)
	}

	return persons, total, nil
}

func (r *PostgresRepository) Commit() error {
	err := r.tx.Commit(context.Background())
	if err != nil {
		r.logger.Errorf("failed to commit transaction %w", err)
		return err
	}
	return nil
}

func (r *PostgresRepository) Rollback() error {
	err := r.tx.Rollback(context.Background())
	if err != nil {
		r.logger.Errorf("failed to rollback transaction %w", err)
		return err
	}
	return nil
}
