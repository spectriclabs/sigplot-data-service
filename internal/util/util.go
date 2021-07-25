package util

// CheckError is a simple error handling
// function that will panic if `e` contains
// an error
func CheckError(e error) {
	if e != nil {
		panic(e)
	}
}
