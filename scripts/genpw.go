// genpw generates a password hash for a given password
package main

import (
	"flag"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

var (
	Password = flag.String("password", "password", "Password to generate hash for")
)

func main() {
	flag.Parse()

	f, _ := bcrypt.GenerateFromPassword([]byte(*Password), bcrypt.DefaultCost)
	fmt.Println(string(f))
}
