package main

import (
	"math/rand"
	"time"
)

type LoanRequest struct {
	UserID    int     `json:"userId"`
	Plafond   float64 `json:"plafond"`
	Duration  float64 `json:"duration"`
	Fees      float64 `json:"fees"`
	StartDate string  `json:"startDate"`
}

type TransferRequest struct {
	ToAccount int `json:"toAccount"`
	Amount    int `json:"amount"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateAccountRequest struct {
	Firstname string `json:"firstName"`
	Lastname  string `json:"lastName"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

type Account struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Firstname string    `json:"firstName"`
	Lastname  string    `json:"lastName"`
	Number    int64     `json:"number"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"createdAt"`
}

type Loan struct {
	ID           int       `json:"id"`
	UserID       int       `json:"userId"`
	Plafond      float64   `json:"amount"`
	LoanDuration float64   `json:"duration"`
	Fees         float64   `json:"fees"`
	StartDate    string    `json:"startDate"`
	CreatedAt    time.Time `json:"createdAt"`
}

func NewLoan(user_id int, plafond float64, duration float64, fees float64, startDate string) (*Loan, error) {
	return &Loan{
		UserID:       user_id,
		Plafond:      plafond,
		LoanDuration: duration,
		Fees:         fees,
		StartDate:    startDate,
		CreatedAt:    time.Now().UTC(),
	}, nil
}

func NewAccount(firstName string, lastName string, email string, password string) (*Account, error) {
	return &Account{
		Firstname: firstName,
		Lastname:  lastName,
		Email:     email,
		Password:  password,
		Number:    int64(rand.Intn(100000)),
		CreatedAt: time.Now().UTC(),
	}, nil
}
