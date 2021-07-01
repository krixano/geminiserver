package main

import (
	"github.com/krixano/ponixserver/src/gemini"
	_ "github.com/krixano/ponixserver/src/migration"
	//"io"
	//"io/ioutil"
	//"os"
	//"database/sql"

	//"github.com/pitr/gig"
	//"github.com/nakagami/firebirdsql"
	//_ "github.com/krixano/ponixserver/src/migration"
	//"github.com/spf13/cobra"
)

func main() {
	/*conn, _ := sql.Open("firebirdsql", firebirdConnectionString)
	defer conn.Close()*/

	gemini.GeminiCommand.Execute();
}
