// Command f4 is the composition root: it exists to build the application and
// run it. Everything it used to hold now lives in internal/app, which is the
// only package allowed to know that every subsystem exists.
package main

import "github.com/unxed/f4/internal/app"

func main() {
	app.Main()
}
