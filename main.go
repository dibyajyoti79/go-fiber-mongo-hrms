package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	DB     *mongo.Database
}

type APIResponse struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var mg MongoInstance

const dbName = "hrms"
const mongoURI = "mongodb://localhost:27017/" + dbName

/*================== EMPLOYEE MODEL ==================*/
type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

/*================== CONNECT TO DATABSE ==================*/
func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	// mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	// mongo.Connect(context.Background(), options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err
	}

	mg = MongoInstance{
		Client: client,
		DB:     db,
	}

	return nil

}

/*================== ERROR HELPER FUNCTION ==================*/
func sendErrorResponse(c *fiber.Ctx, statusCode int, message string) error {
	response := APIResponse{
		Status:  0,
		Message: message,
	}

	return c.Status(statusCode).JSON(response)
}

/*================== MAIN FUNCTION ==================*/
func main() {

	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	app.Get("/employee", func(c *fiber.Ctx) error {

		query := bson.D{{}}

		cursor, err := mg.DB.Collection("employees").Find(c.Context(), query)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		var employees []Employee = make([]Employee, 0)

		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		apiResponse := APIResponse{
			Status:  1,
			Message: "Employee records retrieved successfully",
			Data:    employees,
		}

		return c.Status(200).JSON(apiResponse)

	})

	app.Post("/employee", func(c *fiber.Ctx) error {
		collection := mg.DB.Collection("employees")
		employee := new(Employee)

		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		employee.ID = ""

		insertionResult, err := collection.InsertOne(c.Context(), employee)

		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}

		createdRecord := collection.FindOne(c.Context(), filter)

		createdEmployee := &Employee{}

		createdRecord.Decode(createdEmployee)

		apiResponse := APIResponse{
			Status:  1,
			Message: "Employee record insertd successfully",
			Data:    createdEmployee,
		}

		return c.Status(200).JSON(apiResponse)

	})

	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")

		employeeID, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return c.SendStatus(400)
		}

		employee := new(Employee)
		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		query := bson.D{{Key: "_id", Value: employeeID}}

		update := bson.D{
			{
				Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		err = mg.DB.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				c.SendStatus(400)
			}

			return c.SendStatus(500)
		}

		employee.ID = idParam

		apiResponse := APIResponse{
			Status:  1,
			Message: "Employee record updated successfully",
			Data:    employee,
		}

		return c.Status(200).JSON(apiResponse)

	})

	app.Delete("/employee/:id", func(c *fiber.Ctx) error {

		employeeID, err := primitive.ObjectIDFromHex(c.Params("id"))

		if err != nil {
			// return c.Status(400).JSON(APIResponse{
			// 	Status:  0,
			// 	Message: "Bad Request",
			// })
			return sendErrorResponse(c, 400, "Bad Request")
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		result, err := mg.DB.Collection("employees").DeleteOne(c.Context(), &query)
		if err != nil {
			return c.SendStatus(500)
		}
		if result.DeletedCount < 1 {
			return c.Status(404).JSON(fiber.Map{
				"status":  0,
				"message": "Not Found",
			})
		}

		return c.Status(200).JSON(fiber.Map{
			"status":  1,
			"message": "record deleted",
		})

	})

	log.Fatal(app.Listen(":3000"))

}
