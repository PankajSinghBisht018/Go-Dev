package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
	
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Todo struct {
	ID primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Title string `json:"title,omitempty" bson:"title,omitempty"`
	Completed bool `json:"completed,omitempty" bson:"completed,omitempty"`
}

var client *mongo.Client

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	var err error
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Fatalf("Error disconnecting from MongoDB: %v", err)
		}
	}()

	
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Println("Connected to MongoDB!")

	
	router := mux.NewRouter()
	
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173") 
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})


	router.HandleFunc("/todos", CreateTodoEndpoint).Methods("POST")
	router.HandleFunc("/todos", GetTodosEndpoint).Methods("GET")
	router.HandleFunc("/todos/{id}", GetTodoEndpoint).Methods("GET")
	router.HandleFunc("/todos/{id}", UpdateTodoEndpoint).Methods("PUT")
	router.HandleFunc("/todos/{id}", DeleteTodoEndpoint).Methods("DELETE")


	log.Fatal(http.ListenAndServe(":12345", router))
}


func CreateTodoEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	var todo Todo
	err := json.NewDecoder(request.Body).Decode(&todo)
	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
		response.Write([]byte(`{"message": "Invalid request payload"}`))
		return
	}
	collection := client.Database("todos-app").Collection("todos")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	result, err := collection.InsertOne(ctx, todo)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{"message": "Failed to insert todo"}`))
		return
	}
	response.WriteHeader(http.StatusCreated)
	json.NewEncoder(response).Encode(result.InsertedID)
}

func GetTodosEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	var todos []Todo
	collection := client.Database("todos-app").Collection("todos")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{"message": "Failed to fetch todos"}`))
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var todo Todo
		cursor.Decode(&todo)
		todos = append(todos, todo)
	}
	if err := cursor.Err(); err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{"message": "Cursor error"}`))
		return
	}
	json.NewEncoder(response).Encode(todos)
}

func GetTodoEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	params := mux.Vars(request)
	id, _ := primitive.ObjectIDFromHex(params["id"])
	var todo Todo
	collection := client.Database("todos-app").Collection("todos")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	err := collection.FindOne(ctx, Todo{ID: id}).Decode(&todo)
	if err != nil {
		response.WriteHeader(http.StatusNotFound)
		response.Write([]byte(`{"message": "Todo not found"}`))
		return
	}
	json.NewEncoder(response).Encode(todo)
}


func UpdateTodoEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	params := mux.Vars(request)
	id, _ := primitive.ObjectIDFromHex(params["id"])
	var todo Todo
	err := json.NewDecoder(request.Body).Decode(&todo)
	if err != nil {
		response.WriteHeader(http.StatusBadRequest)
		response.Write([]byte(`{"message": "Invalid request payload"}`))
		return
	}
	collection := client.Database("todos-app").Collection("todos")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"_id": id}
	update := bson.M{"$set": todo}
	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{"message": "Failed to update todo"}`))
		return
	}
	if result.ModifiedCount == 0 {
		response.WriteHeader(http.StatusNotFound)
		response.Write([]byte(`{"message": "Todo not found"}`))
		return
	}
	json.NewEncoder(response).Encode(todo)
}

func DeleteTodoEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	params := mux.Vars(request)
	id, _ := primitive.ObjectIDFromHex(params["id"])
	collection := client.Database("todos-app").Collection("todos")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	result, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{"message": "Failed to delete todo"}`))
		return
	}
	if result.DeletedCount == 0 {
		response.WriteHeader(http.StatusNotFound)
		response.Write([]byte(`{"message": "Todo not found"}`))
		return
	}
	json.NewEncoder(response).Encode("Todo deleted successfully")
}
