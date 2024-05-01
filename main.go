package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	version  string = "0.0.0-dev"
	revision string = "0000000000000000000000000000000000000000"
)

func logRequest(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("begin handling request", "method", r.Method, "host", r.Host, "url", r.URL)
		handler.ServeHTTP(w, r)
		slog.Info("end handling request", "method", r.Method, "host", r.Host, "url", r.URL)
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	slog.SetDefault(logger)

	var showVersion = flag.Bool("version", false, "Show version and exit.")
	var listenAddress = flag.String("listen", ":8000", "Listen address.")

	flag.Parse()

	if *showVersion {
		fmt.Printf("v%s+%s\n", version, revision)
		return
	}

	if flag.NArg() != 0 {
		flag.Usage()
		log.Fatalf("\nERROR You MUST NOT pass any positional arguments")
	}

	http.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
	})

	http.HandleFunc("/", logRequest(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		ctx := r.Context()

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		client, err := connectToMongoDB(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		database := client.Database("counters")   // NB this also creates the database if it does not exist.
		collection := database.Collection("hits") // NB this also creates the collection if it does not exist.

		hitsCounter, err := incrementCounter(ctx, collection)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := map[string]interface{}{
			"appVersion":  fmt.Sprintf("%s+%s", version, revision),
			"hitsCounter": hitsCounter,
		}
		body, err := json.Marshal(data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	}))

	slog.Info("listening", "addr", fmt.Sprintf("http://%s", *listenAddress))

	err := http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		log.Fatalf("Failed to ListenAndServe: %v", err)
	}
}

// see Connecting Programmatically to Amazon DocumentDB at https://docs.aws.amazon.com/documentdb/latest/developerguide/connect_programmatically.html#connect_programmatically-tls_enabled
func connectToMongoDB(ctx context.Context) (*mongo.Client, error) {
	connectionString := ""

	connectionStringSecretID := os.Getenv("DOCDB_EXAMPLE_CONNECTION_STRING_SECRET_ID")
	if connectionStringSecretID != "" {
		connectionStringSecretRegion := os.Getenv("DOCDB_EXAMPLE_CONNECTION_STRING_SECRET_REGION")
		if connectionStringSecretRegion == "" {
			return nil, fmt.Errorf("the DOCDB_EXAMPLE_CONNECTION_STRING_SECRET_REGION environment variable is not set")
		}

		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(connectionStringSecretRegion))
		if err != nil {
			return nil, fmt.Errorf("failed to load default config: %w", err)
		}
		client := secretsmanager.NewFromConfig(cfg)
		secret, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
			SecretId: &connectionStringSecretID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get secret value: %w", err)
		}
		connectionString = *secret.SecretString
	}

	if connectionString == "" {
		connectionString = os.Getenv("DOCDB_EXAMPLE_CONNECTION_STRING")
	}

	if connectionString == "" {
		return nil, fmt.Errorf("the DOCDB_EXAMPLE_CONNECTION_STRING_SECRET_ID or DOCDB_EXAMPLE_CONNECTION_STRING environment variable is not set")
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connectionString))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	// log the existing databases and their collections.
	databaseNames, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	for _, databaseName := range databaseNames {
		log.Printf("mongo database: %s", databaseName)
	}
	for _, databaseName := range databaseNames {
		database := client.Database(databaseName)
		collectionNames, err := database.ListCollectionNames(ctx, bson.M{})
		if err != nil {
			return nil, fmt.Errorf("failed to list database %s collections: %w", databaseName, err)
		}
		for _, collectionName := range collectionNames {
			log.Printf("mongo database %s collection: %s", databaseName, collectionName)
		}
	}

	return client, nil
}

func incrementCounter(ctx context.Context, collection *mongo.Collection) (int, error) {
	filter := bson.M{"_id": "counter"}
	update := bson.M{"$inc": bson.M{"value": 1}}
	result := collection.FindOneAndUpdate(
		ctx,
		filter,
		update,
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After))
	if result.Err() != nil {
		return 0, fmt.Errorf("failed to increment counter: %w", result.Err())
	}
	var counter struct {
		Value int `bson:"value"`
	}
	err := result.Decode(&counter)
	if err != nil {
		return 0, fmt.Errorf("failed to decode increment counter response: %w", result.Err())
	}
	return counter.Value, nil
}
