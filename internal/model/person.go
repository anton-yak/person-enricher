package model

import (
	"errors"
	"fmt"
	"sync"
)

type Person struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Surname     string `json:"surname"`
	Patronymic  string `json:"patronymic,omitempty"`
	Age         uint   `json:"age"`
	Gender      string `json:"gender"`
	Nationality string `json:"nationality"`
}

var ErrPersonNotFound = errors.New("person not found")

// var testPersons []Person = []Person{
// 	{ID: 1, Name: "Dmitriy", Surname: "Ushakov", Patronymic: "Vasilevich"},
// 	{ID: 2, Name: "Anton", Surname: "Yakimenko", Patronymic: "Vladimerovich"},
// 	{ID: 3, Name: "Elon", Surname: "Mask"},
// }

func (p *Person) Validate() error {
	var errs []error

	if p.Name == "" {
		errs = append(errs, errors.New("name can't be empty"))
	}
	if p.Surname == "" {
		errs = append(errs, errors.New("surnname can't be empty"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

type Enricher interface {
	GetAgeByName(string) (uint, error)
	GetGenderByName(string) (string, error)
	GetNationalityByName(string) (string, error)
}

func (p *Person) Enrich(enricher Enricher) error {
	var age uint
	var gender string
	var nationality string

	var ageErr error
	var genderErr error
	var nationalityErr error

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		age, ageErr = enricher.GetAgeByName(p.Name)
		wg.Done()
	}()
	go func() {
		gender, genderErr = enricher.GetGenderByName(p.Name)
		wg.Done()
	}()
	go func() {
		nationality, nationalityErr = enricher.GetNationalityByName(p.Name)
		wg.Done()
	}()

	wg.Wait()

	var errs []error

	if ageErr != nil {
		errs = append(errs, ageErr)
	}
	if genderErr != nil {
		errs = append(errs, genderErr)
	}
	if nationalityErr != nil {
		errs = append(errs, nationalityErr)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to enrich person: %w", errors.Join(errs...))
	}

	p.Age = age
	p.Gender = gender
	p.Nationality = nationality

	return nil
}

type Repository interface {
	InsertPerson(*Person) (uint, error)
	UpdatePerson(*Person) error
	DeletePerson(uint) error
	GetPersonWithLock(uint) (*Person, error)
	GetAllPersons(*Person, *uint, uint) ([]Person, uint, error)
	Commit() error
	Rollback() error
}

func (p *Person) Save(r Repository) error {
	if p.ID != 0 {
		err := r.UpdatePerson(p)
		if err != nil {
			return err
		}
	} else {
		id, err := r.InsertPerson(p)
		if err != nil {
			return err
		}
		p.ID = id
	}

	return nil
}

func (p *Person) Delete(r Repository) error {
	return r.DeletePerson(p.ID)
}

func GetPersonWithLock(r Repository, id uint) (*Person, error) {
	person, err := r.GetPersonWithLock(id)

	if err != nil {
		return nil, fmt.Errorf("person with id %d not found: %w", id, ErrPersonNotFound)
	}
	return person, nil
}

func GetAllPersons(r Repository, person *Person, limit *uint, offset uint) ([]Person, uint, error) {
	persons, total, err := r.GetAllPersons(person, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	return persons, total, nil
}
