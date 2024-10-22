package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jackc/pgx/v4"
)

// DockerImage represents the structure of the Docker Hub image data
type DockerImage struct {
	Name           string   `json:"name"`
	Namespace      string   `json:"namespace"`
	Description    string   `json:"description"`
	PullCount      int      `json:"pull_count"`
	StarCount      int      `json:"star_count"`
	IsPrivate      bool     `json:"is_private"`
	LastUpdated    string   `json:"last_updated"`
	MediaTypes     []string `json:"media_types"`
	ContentTypes   []string `json:"content_types"`
	StorageSize    string
	StorageSizeInt int64        `json:"storage_size"`
	Categories     []Categories `json:"categories"` // 카테고리 정보
	Category       string       `json:"category"`   // 카테고리 이름들
	ImageType      string       // Docker Official Image, Verified Publisher, or Sponsored OSS
	Category1      string       // 첫 번째 카테고리
	Category2      string       // 두 번째 카테고리
	Category3      string       // 세 번째 카테고리
	Category4      string       // 네 번째 카테고리
}

type Categories struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// DBConfig holds PostgreSQL configuration
type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

// DockerHubRepoResponse represents the response from Docker Hub API for repository listings
type DockerHubRepoResponse struct {
	Count   int           `json:"count"`
	Next    string        `json:"next"`
	Results []DockerImage `json:"results"`
}

func main() {
	// Docker Official Image, Verified Publisher, Sponsored OSS의 사용자명 리스트
	verifiedPublishers := []string{
		"datadog", "grafana", "bitnami", "rancher", "amazon",
		"newrelic", "google", "nginxnc", "docker", "kong",
		"hashicorp", "mirantis", "newrelic", "atlassian",
		"jetbrains", "cimg", "intel", "snyk", "redhat", "ksamweb", "circleci",
	}

	sponsoredOSSNamespaces := []string{
		"fluent", "istio", "containerrr", "envoyproxy",
		"jenkins", "linuxserver", "fluxcd", "apache",
		"pihole", "moby", "selenium", "itzg", "alpine",
		"coredns", "nodered", "localstack", "jellyfin",
		"verdaccio", "postgis", "tautulli", "vaultwarden",
		"jupyterhub", "requarks", "eclipse", "gogs", "jupyter",
		"paketobuildpacks", "crossplane", "falcosecurity", "kubernetes",
		"projectcontour",
	}

	// PostgreSQL 설정
	dbConfig := DBConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "master",
		DBName:   "",
	}

	// PostgreSQL 연결
	conn, err := connectToDB(dbConfig)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())

	// Initialize the Resty client
	client := resty.New()

	// Docker Official Images 처리
	officialAPIURL := "https://hub.docker.com/v2/repositories/library/"
	err = fetchAndInsertImages(officialAPIURL, client, conn, "Docker Official Image")
	if err != nil {
		log.Fatalf("Error processing Docker Official Images: %v", err)
	}

	// Verified Publisher들의 이미지 처리
	for _, publisher := range verifiedPublishers {
		apiURL := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/", publisher)
		err = fetchAndInsertImages(apiURL, client, conn, "Verified Publisher")
		if err != nil {
			log.Fatalf("Error processing repositories for %s: %v", publisher, err)
		}
	}

	// Sponsored OSS들의 이미지 처리
	for _, ossNamespace := range sponsoredOSSNamespaces {
		apiURL := fmt.Sprintf("https://hub.docker.com/v2/repositories/%s/", ossNamespace)
		err = fetchAndInsertImages(apiURL, client, conn, "Sponsored OSS")
		if err != nil {
			log.Fatalf("Error processing Sponsored OSS repositories for %s: %v", ossNamespace, err)
		}
	}
}

// fetchAndInsertImages fetches repositories and inserts them into the database
func fetchAndInsertImages(apiURL string, client *resty.Client, conn *pgx.Conn, imageType string) error {
	for apiURL != "" {
		resp, err := makeRequestWithRetry(apiURL, client)
		if err != nil {
			return err
		}

		// Parse the JSON response
		var repoResponse DockerHubRepoResponse
		if err := json.Unmarshal(resp.Body(), &repoResponse); err != nil {
			return fmt.Errorf("Error parsing repositories JSON: %v", err)
		}

		// Insert repositories data into PostgreSQL
		for _, repo := range repoResponse.Results {
			repo.ImageType = imageType // 이미지 유형 추가 (Docker Official Image, Verified Publisher, or Sponsored OSS)

			// 카테고리가 있을 경우 처리
			if len(repo.Categories) > 0 {
				var categoryNames []string
				for _, category := range repo.Categories {
					categoryNames = append(categoryNames, category.Name)
				}
				repo.Category = strings.Join(categoryNames, ", ")

				// 카테고리 최대 4개까지 개별 필드에 할당
				if len(categoryNames) > 0 {
					repo.Category1 = categoryNames[0]
				}
				if len(categoryNames) > 1 {
					repo.Category2 = categoryNames[1]
				}
				if len(categoryNames) > 2 {
					repo.Category3 = categoryNames[2]
				}
				if len(categoryNames) > 3 {
					repo.Category4 = categoryNames[3]
				}
			} else {
				repo.Category = "Uncategorized"
			}

			// 스토리지 크기 변환
			repo.StorageSize = convertStorageSize(repo.StorageSizeInt)
			fmt.Println(repo.StorageSize)

			// Insert into DB
			err = insertRepoToDB(conn, repo)
			if err != nil {
				log.Printf("Error inserting repository data: %v", err)
			} else {
				fmt.Printf("Inserted repository: %s/%s\n", repo.Namespace, repo.Name)
			}
		}

		// Move to the next page (if available)
		apiURL = repoResponse.Next
	}
	return nil
}

// 스토리지 크기를 GB, MB, KB 단위로 변환하는 함수
func convertStorageSize(size int64) string {
	switch {
	case size >= 1<<30: // GB
		return fmt.Sprintf("%.2f GB", float64(size)/(1<<30)) // GB 단위로 변환하여 문자열로 반환
	case size >= 1<<20: // MB
		return fmt.Sprintf("%.2f MB", float64(size)/(1<<20)) // MB 단위로 변환하여 문자열로 반환
	case size >= 1<<10: // KB
		return fmt.Sprintf("%.2f KB", float64(size)/(1<<10)) // KB 단위로 변환하여 문자열로 반환
	default:
		return fmt.Sprintf("%d Bytes", size) // 너무 작은 경우 Bytes로 반환
	}
}

// makeRequestWithRetry makes an HTTP request with retry logic if a 429 status code is received
func makeRequestWithRetry(apiURL string, client *resty.Client) (*resty.Response, error) {
	const maxRetries = 5
	var resp *resty.Response
	var err error

	// retries 변수를 for 루프 외부에 선언
	retries := 0

	for retries = 0; retries < maxRetries; retries++ {
		resp, err = client.R().Get(apiURL)

		// Check if status code is 429 (Too Many Requests)
		if resp != nil && resp.StatusCode() == http.StatusTooManyRequests {
			waitTime := 10 // 기본 대기 시간 (초 단위)

			// 대기 시간 출력 및 재시도
			fmt.Printf("Received 429 Too Many Requests, retrying after %d seconds...\n", waitTime)
			time.Sleep(time.Duration(waitTime) * time.Second)
			continue
		}

		// 정상 응답을 받았거나 다른 오류 발생 시 종료
		if err == nil && resp.StatusCode() != http.StatusTooManyRequests {
			break
		}
	}

	// 최대 재시도 횟수 초과 시 오류 반환
	if retries == maxRetries {
		return nil, fmt.Errorf("max retries exceeded for URL: %s", apiURL)
	}

	return resp, err
}

// connectToDB initializes the connection to PostgreSQL using pgx
func connectToDB(config DBConfig) (*pgx.Conn, error) {
	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		config.User, config.Password, config.Host, config.Port, config.DBName)

	// pgx를 사용하여 DB에 연결
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// insertRepoToDB inserts the Docker repository data into PostgreSQL
func insertRepoToDB(conn *pgx.Conn, repo DockerImage) error {
	query := `
		INSERT INTO docker_images (name, namespace, description, pull_count, star_count, is_private, last_updated, media_types, content_types, storage_size, category1, category2, category3, category4, image_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (name, namespace) DO NOTHING;
	`
	mediaTypes := strings.Join(repo.MediaTypes, ", ")
	contentTypes := strings.Join(repo.ContentTypes, ", ")

	_, err := conn.Exec(context.Background(), query,
		repo.Name, repo.Namespace, repo.Description, repo.PullCount, repo.StarCount, repo.IsPrivate, repo.LastUpdated, mediaTypes, contentTypes, repo.StorageSize, repo.Category1, repo.Category2, repo.Category3, repo.Category4, repo.ImageType)
	return err
}
