package main

import "fmt"

// No parameters and no results.
func Dummy() {}

// Parameter has anonymous struct type.
func GreetPerson(person struct {
	name, surname string
	age           int
}, id int) {
	fmt.Println("Hi ", person, id)
}

// Function where multiple parameters share type and which returns multiple results.
func SafeDivide(x, y int) (result int, ok bool) {
	if y == 0 {
		return 0, false
	}
	return x / y, true
}

// Generic function.
func Identity[T any](x T) T {
	return x
}

// Generic struct.
type KeyId[K comparable, I ~int | ~uint] struct {
	Key K
	Id  I
}
