package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func main() {
	engine := gin.Default()

	engine.GET("/", func(ctx *gin.Context) {
		fmt.Println("我是中间件")
	}, func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello world")
		fmt.Println("hello world")
	}, func(ctx *gin.Context) {
		fmt.Println("我也是中间将")
	})

	engine.GET("/:name", BindingExample)

	engine.GET("/:form", func(ctx *gin.Context) {
		ctx.PostForm("key")
		ctx.Query("name")
		ctx.Params.Get("form")
		ctx.Param("form")
		ctx.FormFile("file")
	})

	engine.GET("/binding", func(ctx *gin.Context) {
		var request bindingRequest
		ctx.Bind(&request)
		ctx.BindHeader(&request)
		ctx.BindJSON(&request)
		ctx.BindQuery(&request)
		ctx.BindUri(&request)
		ctx.BindXML(&request)
		ctx.BindYAML(&request)
		// ctx.BindWith(&request, binding.Binding)
		ctx.ShouldBind(&request)
		// ctx.ShouldBindBodyWith(&request, binding.Binding)
		ctx.ShouldBindHeader(&request)
		ctx.ShouldBindJSON(&request)
		ctx.ShouldBindQuery(&request)
		ctx.ShouldBindUri(&request)
		ctx.ShouldBindXML(&request)
		ctx.ShouldBindYAML(&request)
	})

	engine.GET("/render", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello world")
		ctx.JSON(http.StatusOK, gin.H{"hello": "world"})
		ctx.JSONP(http.StatusOK, gin.H{"hello": "world"})
		// ctx.HTML()
		// ctx.File()
	})

	log.Fatal(engine.Run(":8080"))
}

type bindingRequest struct {
	Name string `json:"name"`
}

func BindingExample(ctx *gin.Context) {
	name, exists := ctx.Params.Get("name")
	if !exists {
		ctx.String(http.StatusOK, "not found")
		return
	}
	ctx.String(http.StatusOK, name)
}
