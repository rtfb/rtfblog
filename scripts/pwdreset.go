package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/howeyc/gopass"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func EncryptBcrypt(passwd string) (hash string, err error) {
	hashBytes, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	hash = string(hashBytes)
	return
}

func init_db(connString, uname, passwd, fullname, email, www string) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer db.Close()
	//stmt, _ := db.Prepare(`insert into author(id, disp_name, passwd, full_name, email, www)
	//values($1, $2, $3, $4, $5, $6)`)
	stmt, _ := db.Prepare(`update author set
	disp_name=$1, passwd=$2, full_name=$3, email=$4, www=$5
	where id=1`)
	defer stmt.Close()
	passwdHash, err := EncryptBcrypt(passwd)
	if err != nil {
		fmt.Printf("Error in Encrypt(): %s\n", err)
		return
	}
	fmt.Printf("Updating user ID=1...\n")
	fmt.Printf("dbstr: %q\nuname: %q\npasswd: %q\nhash: %q\nfullname: %q\nemail: %q\nwww: %q\n",
		connString, uname, "***", passwdHash, fullname, email, www)
	stmt.Exec(uname, passwdHash, fullname, email, www)
}

func main() {
	dbFile := os.Getenv("RTFBLOG_DB_TEST_URL")
	uname := "rtfb"
	fmt.Printf("New password: ")
	passwd := gopass.GetPasswd()
	fmt.Printf("Confirm: ")
	passwd2 := gopass.GetPasswd()
	if string(passwd2) != string(passwd) {
		panic("Passwords do not match")
	}
	fullname := "Vytautas Šaltenis"
	email := "vytas@rtfb.lt"
	www := "http://rtfb.lt/"
	init_db(dbFile, uname, string(passwd), fullname, email, www)
}
