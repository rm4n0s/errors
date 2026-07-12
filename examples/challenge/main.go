package main

import (
	"fmt"
	"math/rand/v2"

	"github.com/rm4n0s/errors"
)

var errBankAccountEmpty = "AccountIsEmpty"
var errInvestmentLost = "InvestmentLost"

func f1() error {
	n := rand.IntN(9) + 1
	if n%2 == 0 {
		return errors.New(errBankAccountEmpty, "account is empty")
	}
	return errors.New(errInvestmentLost, "investment is lost")
}

func f2() error {
	return f1()

}

func f3() error {
	return f1()
}

func f4() error {
	n := rand.IntN(9) + 1
	if n%2 == 0 {
		return f2()
	}
	return f3()
}

func main() {
	err := f4()
	fmt.Println(err)

	terr, _ := errors.FromError(err)
	fmt.Printf("%#v \n", terr)
	fmt.Println(terr.Route())

	if terr.HasRoute("f4->f3->f1." + errInvestmentLost) {
		fmt.Println("The money in your account didn't do well")
	} else if terr.HasRoute("f4->f2->f1." + errBankAccountEmpty) {
		fmt.Println("Aand it's gone")
	} else {
		fmt.Println("This line is for bank members only")
	}
}
