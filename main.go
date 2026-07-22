package main

import (
	"os"

	"github.com/engigu/baihu-panel/cmd"
)

// @title Baihu Panel API
// @version 1.0
// @description Baihu Panel OpenAPI Server documentation.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:8052
// @BasePath /open2api/v1
// @query.collection.format multi
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the API token.

func main() {
	cmd.InitHandlers()
	cmd.Execute(os.Args)
}
