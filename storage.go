package main

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type Storage interface {
	CreateAccount(*Account) error
	UpdateAccount(*Account) error
	GetAccountById(int) (*Account, error)
	GetAccounts() ([]*Account, error)
	DeleteAccount(int) error
	LoginAccount(string, string) (*Account, error)
	CreateLoan(*Loan) error
}

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore() (*PostgresStore, error) {
	conSstr := "user=postgres dbname=postgres password=gobank sslmode=disable"
	db, err := sql.Open("postgres", conSstr)
	if err != nil {

		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) init() error {
	s.createAccountTable()
	s.loanTable()
	return nil
}

func (s *PostgresStore) createAccountTable() error {
	query := `CREATE TABLE IF NOT EXISTS account (
		id serial primary key,
		first_name varchar(50),
		last_name varchar(50),
		email varchar NOT NULL UNIQUE,
		password varchar,
		number serial,
		balance serial,
		created_at timestamp
		)`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) loanTable() error {
	query := `CREATE TABLE IF NOT EXISTS loan (
		id serial primary key,
		user_id serial NOT NULL references account(id),
		plafond serial NOT NULL,
		duration varchar NOT NULL,
		fees varchar,
		start_date timestamp,
		created_at timestamp
		)`
	_, err := s.db.Exec(query)
	return err
}

func (s *PostgresStore) CreateAccount(acc *Account) error {
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(acc.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	query := `INSERT INTO account
	(first_name, last_name, email, password, number, balance, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = s.db.Exec(
		query,
		acc.Firstname,
		acc.Lastname,
		acc.Email,
		string(hashedPass),
		acc.Number,
		acc.Balance,
		acc.CreatedAt,
	)

	return err
}

func (s *PostgresStore) CreateLoan(lo *Loan) error {
	query := `INSERT INTO loan
	(user_id, plafond, duration, fees, start_date, created_at)
	VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := s.db.Exec(
		query,
		lo.UserID,
		lo.Plafond,
		lo.LoanDuration,
		lo.Fees,
		lo.StartDate,
		lo.CreatedAt,
	)

	return err
}

func (s *PostgresStore) UpdateAccount(*Account) error {

	return nil
}

func (s *PostgresStore) DeleteAccount(id int) error {
	_, err := s.db.Query("delete from account where id=$1", id)
	return err
}

func (s *PostgresStore) GetAccountById(id int) (*Account, error) {
	rows, err := s.db.Query("select * from account where id=$1", id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		return ScanIntoAccount(rows)
	}
	return nil, fmt.Errorf("account with id %d not found", id)
}

func (s *PostgresStore) LoginAccount(email string, password string) (*Account, error) {
	rows, err := s.db.Query("select * from account where email=$1", email)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		account, err := ScanIntoAccount(rows)
		if err != nil {
			return nil, err
		}
		err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password))
		if err != nil {
			return nil, fmt.Errorf("incorrect password")
		}
		return account, nil
	}
	return nil, fmt.Errorf("account with email %s not found", email)
}

func (s *PostgresStore) GetAccounts() ([]*Account, error) {
	rows, err := s.db.Query("select * from account")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []*Account{}
	for rows.Next() {
		account, err := ScanIntoAccount(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func ScanIntoAccount(rows *sql.Rows) (*Account, error) {
	account := new(Account)
	err := rows.Scan(
		&account.ID,
		&account.Firstname,
		&account.Lastname,
		&account.Email,
		&account.Password,
		&account.Number,
		&account.Balance,
		&account.CreatedAt,
	)
	return account, err

}
