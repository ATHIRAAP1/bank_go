package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/dixonwille/wmenu/v5"
	_ "github.com/mattn/go-sqlite3"
)

type person struct {
	id         int
	first_name string
	last_name  string
	email      string
	ip_address string
}

type account struct {
	id       int
	personID int
	balance  float64
}

func main() {
	// Open the database connection
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check if the connection is valid
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	// Create the tables if they do not exist
	createTables(db)

	// Create the menu
	menu := wmenu.NewMenu("What would you like to do?")

	menu.Action(func(opts []wmenu.Opt) error {
		handleFunc(db, opts)
		return nil
	})

	menu.Option("Add a new Person", 0, true, nil)
	menu.Option("Create an Account", 1, false, nil)
	menu.Option("Deposit Money", 2, false, nil)
	menu.Option("Withdraw Money", 3, false, nil)
	menu.Option("View Account", 4, false, nil)
	menuerr := menu.Run()

	if menuerr != nil {
		log.Fatal(menuerr)
	}
}

func createTables(db *sql.DB) {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS people (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		first_name TEXT,
		last_name TEXT,
		email TEXT,
		ip_address TEXT
	);

	CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		person_id INTEGER,
		balance REAL,
		FOREIGN KEY(person_id) REFERENCES people(id)
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER,
		amount REAL,
		transaction_type TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(account_id) REFERENCES accounts(id)
	);
	`)
	if err != nil {
		log.Fatal(err)
	}
}

func handleFunc(db *sql.DB, opts []wmenu.Opt) {
	switch opts[0].Value {
	case 0:
		addPerson(db)
	case 1:
		createAccount(db)
	case 2:
		depositMoney(db)
	case 3:
		withdrawMoney(db)
	case 4:
		viewAccount(db)
	}
}

func addPerson(db *sql.DB) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter a first name: ")
	firstName, _ := reader.ReadString('\n')
	firstName = firstName[:len(firstName)-1] // Trim the newline character

	fmt.Print("Enter a last name: ")
	lastName, _ := reader.ReadString('\n')
	lastName = lastName[:len(lastName)-1] // Trim the newline character

	fmt.Print("Enter an email address: ")
	email, _ := reader.ReadString('\n')
	email = email[:len(email)-1] // Trim the newline character

	fmt.Print("Enter an IP address: ")
	ipAddress, _ := reader.ReadString('\n')
	ipAddress = ipAddress[:len(ipAddress)-1] // Trim the newline character

	newPerson := person{
		first_name: firstName,
		last_name:  lastName,
		email:      email,
		ip_address: ipAddress,
	}

	stmt, err := db.Prepare("INSERT INTO people (first_name, last_name, email, ip_address) VALUES (?, ?, ?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(newPerson.first_name, newPerson.last_name, newPerson.email, newPerson.ip_address)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Added %v %v \n", newPerson.first_name, newPerson.last_name)
}

func createAccount(db *sql.DB) {
	var personID int
	fmt.Print("Enter the person ID for the new account: ")
	fmt.Scan(&personID)

	stmt, err := db.Prepare("INSERT INTO accounts (person_id, balance) VALUES (?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(personID, 0.0)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Account created successfully.")
}

func depositMoney(db *sql.DB) {
	var accountID int
	var amount float64
	fmt.Print("Enter the account ID to deposit money: ")
	fmt.Scan(&accountID)
	fmt.Print("Enter the amount to deposit: ")
	fmt.Scan(&amount)

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare("UPDATE accounts SET balance = balance + ? WHERE id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(amount, accountID)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	stmt, err = tx.Prepare("INSERT INTO transactions (account_id, amount, transaction_type) VALUES (?, ?, 'deposit')")
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(accountID, amount)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	tx.Commit()

	fmt.Println("Deposit successful.")
}

func withdrawMoney(db *sql.DB) {
	var accountID int
	var amount float64
	fmt.Print("Enter the account ID to withdraw money: ")
	fmt.Scan(&accountID)
	fmt.Print("Enter the amount to withdraw: ")
	fmt.Scan(&amount)

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare("UPDATE accounts SET balance = balance - ? WHERE id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(amount, accountID)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	stmt, err = tx.Prepare("INSERT INTO transactions (account_id, amount, transaction_type) VALUES (?, ?, 'withdrawal')")
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(accountID, amount)
	if err != nil {
		tx.Rollback()
		log.Fatal(err)
	}

	tx.Commit()

	fmt.Println("Withdrawal successful.")
}

func viewAccount(db *sql.DB) {
	var accountID int
	fmt.Print("Enter the account ID to view: ")
	fmt.Scan(&accountID)

	row := db.QueryRow("SELECT id, person_id, balance FROM accounts WHERE id = ?", accountID)
	var acc account
	err := row.Scan(&acc.id, &acc.personID, &acc.balance)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Account not found.")
		} else {
			log.Fatal(err)
		}
		return
	}

	fmt.Printf("Account ID: %d\nPerson ID: %d\nBalance: %.2f\n", acc.id, acc.personID, acc.balance)

	rows, err := db.Query("SELECT id, amount, transaction_type, created_at FROM transactions WHERE account_id = ?", acc.id)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Transactions:")
	for rows.Next() {
		var id int
		var amount float64
		var transactionType string
		var createdAt string
		err := rows.Scan(&id, &amount, &transactionType, &createdAt)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %d | Amount: %.2f | Type: %s | Date: %s\n", id, amount, transactionType, createdAt)
	}
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
}
