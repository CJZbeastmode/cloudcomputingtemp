package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Defines a "model" that we can use to communicate with the
// frontend or the database
type BookStore struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	BookName   string             `bson:"BookName"`
	BookAuthor string             `bson:"BookAuthor"`
	BookISBN   string             `bson:"BookISBN"`
	BookPages  int                `bson:"BookPages"`
	BookYear   int                `bson:"BookYear"`
}

// Wraps the "Template" struct to associate a necessary method
// to determine the rendering procedure
type Template struct {
	tmpl *template.Template
}

// Preload the available templates for the view folder.
// This builds a local "database" of all available "blocks"
// to render upon request, i.e., replace the respective
// variable or expression.
// For more on templating, visit https://jinja.palletsprojects.com/en/3.0.x/templates/
// to get to know more about templating
// You can also read Golang's documentation on their templating
// https://pkg.go.dev/text/template
func loadTemplates() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("views/*.html")),
	}
}

// Method definition of the required "Render" to be passed for the Rendering
// engine.
// Contraire to method declaration, such syntax defines methods for a given
// struct. "Interfaces" and "structs" can have methods associated with it.
// The difference lies that interfaces declare methods whether struct only
// implement them, i.e., only define them. Such differentiation is important
// for a compiler to ensure types provide implementations of such methods.
func (t *Template) Render(w io.Writer, name string, data interface{}, ctx echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

// Here we make sure the connection to the database is correct and initial
// configurations exists. Otherwise, we create the proper database and collection
// we will store the data.
// To ensure correct management of the collection, we create a return a
// reference to the collection to always be used. Make sure if you create other
// files, that you pass the proper value to ensure communication with the
// database
// More on what bson means: https://www.mongodb.com/docs/drivers/go/current/fundamentals/bson/
func prepareDatabase(client *mongo.Client, dbName string, collecName string) (*mongo.Collection, error) {
	db := client.Database(dbName)

	names, err := db.ListCollectionNames(context.TODO(), bson.D{{}})
	if err != nil {
		return nil, err
	}
	if !slices.Contains(names, collecName) {
		cmd := bson.D{{"create", collecName}}
		var result bson.M
		if err = db.RunCommand(context.TODO(), cmd).Decode(&result); err != nil {
			log.Fatal(err)
			return nil, err
		}
	}

	coll := db.Collection(collecName)
	return coll, nil
}

// Here we prepare some fictional data and we insert it into the database
// the first time we connect to it. Otherwise, we check if it already exists.
func prepareData(client *mongo.Client, coll *mongo.Collection) {
	startData := []BookStore{
		{
			BookName:   "The Vortex",
			BookAuthor: "José Eustasio Rivera",
			BookISBN:   "958-30-0804-4",
			BookPages:  292,
			BookYear:   1924,
		},
		{
			BookName:   "Frankenstein",
			BookAuthor: "Mary Shelley",
			BookISBN:   "978-3-649-64609-9",
			BookPages:  280,
			BookYear:   1818,
		},
		{
			BookName:   "The Black Cat",
			BookAuthor: "Edgar Allan Poe",
			BookISBN:   "978-3-99168-238-7",
			BookPages:  280,
			BookYear:   1843,
		},
	}

	// This syntax helps us iterate over arrays. It behaves similar to Python
	// However, range always returns a tuple: (idx, elem). You can ignore the idx
	// by using _.
	// In the topic of function returns: sadly, there is no standard on return types from function. Most functions
	// return a tuple with (res, err), but this is not granted. Some functions
	// might return a ret value that includes res and the err, others might have
	// an out parameter.
	for _, book := range startData {
		cursor, err := coll.Find(context.TODO(), book)
		var results []BookStore
		if err = cursor.All(context.TODO(), &results); err != nil {
			panic(err)
		}
		if len(results) > 1 {
			log.Fatal("more records were found")
		} else if len(results) == 0 {
			result, err := coll.InsertOne(context.TODO(), book)
			if err != nil {
				panic(err)
			} else {
				fmt.Printf("%+v\n", result)
			}

		} else {
			for _, res := range results {
				cursor.Decode(&res)
				fmt.Printf("%+v\n", res)
			}
		}
	}
}

// Generic method to perform "SELECT * FROM BOOKS" (if this was SQL, which
// it is not :D ), and then we convert it into an array of map. In Golang, you
// define a map by writing map[<key type>]<value type>{<key>:<value>}.
// interface{} is a special type in Golang, basically a wildcard...
func findAllBooks(coll *mongo.Collection) []map[string]interface{} {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"ID":         res.ID.Hex(),
			"BookName":   res.BookName,
			"BookAuthor": res.BookAuthor,
			"BookISBN":   res.BookISBN,
			"BookPages":  res.BookPages,
		})
	}

	return ret
}

func findAllBooksWithYears(coll *mongo.Collection) []map[string]interface{} {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"ID":         res.ID,
			"BookName":   res.BookName,
			"BookAuthor": res.BookAuthor,
			"BookISBN":   res.BookISBN,
			"BookPages":  res.BookPages,
			"BookYear":   res.BookYear,
		})
	}

	return ret
}

func findAllAuthors(coll *mongo.Collection) []string {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []string
	for _, res := range results {
		ret = append(ret, res.BookAuthor)
	}

	return ret
}

func findAllYears(coll *mongo.Collection) []int {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []int
	for _, res := range results {
		ret = append(ret, res.BookYear)
	}

	return ret
}

func addBook(coll *mongo.Collection, book BookStore) int {
	filter := bson.M{"BookName": book.BookName,
		"BookAuthor": book.BookAuthor,
		"BookISBN":   book.BookISBN,
		"BookPages":  book.BookPages,
		"BookYear":   book.BookYear}

	var results []*BookStore
	cursor, err := coll.Find(context.TODO(), filter)
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.TODO())

	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	for cursor.Next(context.TODO()) {
		var elem BookStore
		err := cursor.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}
		results = append(results, &elem)
	}

	fmt.Println("State 1")
	if len(results) > 0 {
		return 304
	}

	fmt.Println("State 2")
	_, err = coll.InsertOne(context.TODO(), book)
	if err != nil {
		return 304
	}
	return 200
}

func updateBook(coll *mongo.Collection, book BookStore) int {
	update := bson.M{"$set": book}
	filter := bson.M{"_id": book.ID}
	res, err := coll.UpdateOne(context.TODO(), filter, update)
	fmt.Println(res.ModifiedCount)
	if err != nil || res.ModifiedCount == 0 {
		return 304
	}
	return 200
}

func deleteBook(coll *mongo.Collection, id primitive.ObjectID) int {
	filter := bson.M{"_id": id}
	result, err := coll.DeleteOne(context.TODO(), filter)
	if err != nil || result.DeletedCount == 0 {
		return 304
	}
	return 200
}

func main() {
	// Connect to the database. Such defer keywords are used once the local
	// context returns; for this case, the local context is the main function
	// By user defer function, we make sure we don't leave connections
	// dangling despite the program crashing. Isn't this nice? :D
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO: make sure to pass the proper username, password, and port
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://mongodb:testmongo@localhost:27017/"))

	// This is another way to specify the call of a function. You can define inline
	// functions (or anonymous functions, similar to the behavior in Python)
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	// You can use such name for the database and collection, or come up with
	// one by yourself!
	coll, err := prepareDatabase(client, "exercise-1", "information")
	prepareData(client, coll)

	// Here we prepare the server
	e := echo.New()

	// Define our custom renderer
	e.Renderer = loadTemplates()

	// Log the requests. Please have a look at echo's documentation on more
	// middleware
	e.Use(middleware.Logger())

	e.Static("/css", "css")

	// Endpoint definition. Here, we divided into two groups: top-level routes
	// starting with /, which usually serve webpages. For our RESTful endpoints,
	// we prefix the route with /api to indicate more information or resources
	// are available under such route.
	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index", nil)
	})

	e.GET("/books", func(c echo.Context) error {
		books := findAllBooks(coll)
		return c.Render(200, "book-table", books)
	})

	e.GET("/authors", func(c echo.Context) error {
		authors := findAllAuthors(coll)
		return c.Render(200, "author", authors)
	})

	e.GET("/years", func(c echo.Context) error {
		years := findAllYears(coll)
		return c.Render(200, "year", years)
	})

	e.GET("/search", func(c echo.Context) error {
		return c.Render(200, "search-bar", nil)
	})

	e.GET("/create", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	e.GET("/api/books", func(c echo.Context) error {
		books := findAllBooksWithYears(coll)
		var ret []map[string]interface{}
		for _, res := range books {
			ret = append(ret, map[string]interface{}{
				"id":     res["ID"],
				"name":   res["BookName"],
				"author": res["BookAuthor"],
				"pages":  res["BookPages"],
				"year":   res["BookYear"],
				"isbn":   res["BookISBN"]})
		}
		return c.JSON(http.StatusOK, ret)
	})

	e.POST("/api/books", func(c echo.Context) error {
		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		defer c.Request().Body.Close()

		var data map[string]interface{}
		err = json.Unmarshal(body, &data)

		if err != nil {
			return err
		}

		var name string = data["name"].(string)
		var author string = data["author"].(string)
		var pages int = int(data["pages"].(float64))
		var year int = int(data["year"].(float64))
		var isbn string = ""
		if _, ok := data["isbn"]; ok {
			isbn = data["isbn"].(string)
		}

		book := BookStore{
			BookName:   name,
			BookAuthor: author,
			BookPages:  pages,
			BookYear:   year,
			BookISBN:   isbn,
		}

		ret := addBook(coll, book)
		if ret == 304 {
			return c.JSON(ret, "bad")
		}
		return c.JSON(ret, book)
	})

	e.PUT("/api/books", func(c echo.Context) error {
		body, err := ioutil.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		defer c.Request().Body.Close()

		var data map[string]interface{}
		err = json.Unmarshal(body, &data)

		if err != nil {
			return err
		}

		var name string = data["name"].(string)
		var author string = data["author"].(string)
		var pages int = int(data["pages"].(float64))
		var year int = int(data["year"].(float64))
		id, err := primitive.ObjectIDFromHex(data["id"].(string))
		var isbn string = ""
		if _, ok := data["isbn"]; ok {
			isbn = data["isbn"].(string)
		}

		book := BookStore{
			BookName:   name,
			BookAuthor: author,
			BookPages:  pages,
			BookYear:   year,
			BookISBN:   isbn,
			ID:         id,
		}

		ret := updateBook(coll, book)
		return c.JSON(ret, book)
	})

	e.DELETE("/api/books/:id", func(c echo.Context) error {
		id := c.Param("id")
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return err
		}
		ret := deleteBook(coll, objID)
		return c.JSON(ret, objID)
	})

	e.Logger.Fatal(e.Start(":3030"))
}
