package cmd

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

func server() {

	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) {
		c.String(200, "Hello, Geektutu")
	})
	r.Run(":8080")
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Print the version number of Hugo",
	Long:  `All software has versions. This is Hugo's`,
	Run: func(cmd *cobra.Command, args []string) {
		server()
	},
}
