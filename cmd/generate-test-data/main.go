package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kungfusheep/glint"
)

// User represents a user in our test data
type User struct {
	ID         int               `json:"id" glint:"id"`
	Username   string            `json:"username" glint:"username"`
	Email      string            `json:"email" glint:"email"`
	FirstName  string            `json:"firstName" glint:"firstName"`
	LastName   string            `json:"lastName" glint:"lastName"`
	Age        int               `json:"age" glint:"age"`
	Active     bool              `json:"active" glint:"active"`
	Score      int               `json:"score" glint:"score"`
	Tags       []string          `json:"tags" glint:"tags"`
	Preferences UserPreferences  `json:"preferences" glint:"preferences"`
	Address    *Address          `json:"address" glint:"address"`
}

type UserPreferences struct {
	Theme         string `json:"theme" glint:"theme"`
	Notifications bool   `json:"notifications" glint:"notifications"`
	Language      string `json:"language" glint:"language"`
	Timezone      string `json:"timezone" glint:"timezone"`
}

type Address struct {
	Street   string `json:"street" glint:"street"`
	City     string `json:"city" glint:"city"`
	Country  string `json:"country" glint:"country"`
	ZipCode  string `json:"zipCode" glint:"zipCode"`
}

// Post represents a blog post
type Post struct {
	ID         int       `json:"id" glint:"id"`
	AuthorID   int       `json:"authorId" glint:"authorId"`
	Title      string    `json:"title" glint:"title"`
	Content    string    `json:"content" glint:"content"`
	Published  bool      `json:"published" glint:"published"`
	Timestamp  string    `json:"timestamp" glint:"timestamp"`
	Likes      int       `json:"likes" glint:"likes"`
	Shares     int       `json:"shares" glint:"shares"`
	Categories []string  `json:"categories" glint:"categories"`
	Comments   []Comment `json:"comments" glint:"comments"`
}

type Comment struct {
	ID        int    `json:"id" glint:"id"`
	AuthorID  int    `json:"authorId" glint:"authorId"`
	Text      string `json:"text" glint:"text"`
	Timestamp string `json:"timestamp" glint:"timestamp"`
	Likes     int    `json:"likes" glint:"likes"`
}

// Analytics represents site analytics
type Analytics struct {
	TotalUsers           int            `json:"totalUsers" glint:"totalUsers"`
	TotalPosts           int            `json:"totalPosts" glint:"totalPosts"`
	AveragePostsPerUser  float64        `json:"averagePostsPerUser" glint:"averagePostsPerUser"`
	TopCategories        []string       `json:"topCategories" glint:"topCategories"`
	MonthlyStats         []MonthlyStat  `json:"monthlyStats" glint:"monthlyStats"`
}

type MonthlyStat struct {
	Month      int     `json:"month" glint:"month"`
	Users      int     `json:"users" glint:"users"`
	Posts      int     `json:"posts" glint:"posts"`
	Engagement float64 `json:"engagement" glint:"engagement"`
}

// MediumDataset contains users only
type MediumDataset struct {
	Users []User `json:"users" glint:"users"`
}

// LargeDataset contains users and posts
type LargeDataset struct {
	Users []User `json:"users" glint:"users"`
	Posts []Post `json:"posts" glint:"posts"`
}

// HugeDataset contains everything
type HugeDataset struct {
	Users     []User    `json:"users" glint:"users"`
	Posts     []Post    `json:"posts" glint:"posts"`
	Analytics Analytics `json:"analytics" glint:"analytics"`
	Metadata  Metadata  `json:"metadata" glint:"metadata"`
}

type Metadata struct {
	Version     string `json:"version" glint:"version"`
	Generated   string `json:"generated" glint:"generated"`
	UserCount   int    `json:"userCount" glint:"userCount"`
	PostCount   int    `json:"postCount" glint:"postCount"`
	Description string `json:"description" glint:"description"`
}

func generateUsers(count int) []User {
	users := make([]User, count)
	for i := 0; i < count; i++ {
		users[i] = User{
			ID:        i + 1,
			Username:  fmt.Sprintf("user%d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			FirstName: fmt.Sprintf("FirstName%d", i),
			LastName:  fmt.Sprintf("LastName%d", i),
			Age:       18 + (i % 50),
			Active:    i%2 == 0,
			Score:     1000 + i*10,
			Tags:      []string{fmt.Sprintf("tag%d", i%10), fmt.Sprintf("category%d", i%5), fmt.Sprintf("level%d", i%3)},
			Preferences: UserPreferences{
				Theme:         map[bool]string{true: "dark", false: "light"}[i%2 == 0],
				Notifications: true,
				Language:      "en",
				Timezone:      "UTC",
			},
			Address: &Address{
				Street:  fmt.Sprintf("%d Main St", i),
				City:    fmt.Sprintf("City%d", i%10),
				Country: "US",
				ZipCode: fmt.Sprintf("1000%d", i),
			},
		}
	}
	return users
}

func generatePosts(count int, userCount int) []Post {
	posts := make([]Post, count)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	
	for i := 0; i < count; i++ {
		content := fmt.Sprintf("This is the content of post %d. It contains some detailed information about the topic and provides value to readers. The post discusses various aspects of the subject matter and includes relevant examples.", i)
		
		comments := make([]Comment, 3)
		for j := 0; j < 3; j++ {
			comments[j] = Comment{
				ID:        j + 1,
				AuthorID:  (j*7+i)%userCount + 1,
				Text:      fmt.Sprintf("Comment %d on post %d", j, i),
				Timestamp: baseTime.Add(time.Duration(i*24+j) * time.Hour).Format(time.RFC3339),
				Likes:     (j * 11) % 50,
			}
		}
		
		posts[i] = Post{
			ID:         i + 1,
			AuthorID:   (i % userCount) + 1,
			Title:      fmt.Sprintf("Post Title %d", i),
			Content:    content,
			Published:  i%3 != 0,
			Timestamp:  baseTime.Add(time.Duration(i*24) * time.Hour).Format(time.RFC3339),
			Likes:      i * 13 % 1000,
			Shares:     i * 7 % 100,
			Categories: []string{fmt.Sprintf("cat%d", i%10), fmt.Sprintf("topic%d", i%5)},
			Comments:   comments,
		}
	}
	return posts
}

func writeGlintFile[T any](filename string, data T) error {
	// Create encoder for the type
	encoder := glint.NewEncoder[T]()
	
	// Create a buffer and marshal the data
	buffer := glint.NewBufferFromPool()
	defer buffer.ReturnToPool()
	
	encoder.Marshal(&data, buffer)
	
	// The encoder produces a complete document with schema included
	encoded := buffer.Bytes
	
	if err := os.WriteFile(filename, encoded, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	fmt.Printf("✓ Created %s (%d bytes)\n", filename, len(encoded))
	return nil
}

func writeJSONFile[T any](filename string, data T) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	
	fmt.Printf("✓ Created %s (%d bytes)\n", filename, len(jsonData))
	return nil
}

func main() {
	fmt.Println("Generating Glint test data...")
	
	// Create output directory if it doesn't exist
	outputDir := "../../cmd/client-ts/test"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	
	// Generate medium dataset (100 users)
	mediumData := MediumDataset{
		Users: generateUsers(100),
	}
	if err := writeGlintFile(fmt.Sprintf("%s/medium-go.glint", outputDir), mediumData); err != nil {
		log.Printf("Error writing medium dataset: %v", err)
	}
	if err := writeJSONFile(fmt.Sprintf("%s/medium-go.json", outputDir), mediumData); err != nil {
		log.Printf("Error writing medium JSON: %v", err)
	}
	
	// Generate large dataset (100 users, 200 posts)
	largeData := LargeDataset{
		Users: generateUsers(100),
		Posts: generatePosts(200, 100),
	}
	if err := writeGlintFile(fmt.Sprintf("%s/large-go.glint", outputDir), largeData); err != nil {
		log.Printf("Error writing large dataset: %v", err)
	}
	if err := writeJSONFile(fmt.Sprintf("%s/large-go.json", outputDir), largeData); err != nil {
		log.Printf("Error writing large JSON: %v", err)
	}
	
	// Generate huge dataset (300 users, 600 posts, with analytics)
	hugeUsers := generateUsers(300)
	hugePosts := generatePosts(600, 300)
	
	monthlyStats := make([]MonthlyStat, 12)
	for i := 0; i < 12; i++ {
		monthlyStats[i] = MonthlyStat{
			Month:      i + 1,
			Users:      100 + i*50,
			Posts:      500 + i*100,
			Engagement: 0.5 + float64(i)*0.03,
		}
	}
	
	hugeData := HugeDataset{
		Users: hugeUsers,
		Posts: hugePosts,
		Analytics: Analytics{
			TotalUsers:          len(hugeUsers),
			TotalPosts:          len(hugePosts),
			AveragePostsPerUser: float64(len(hugePosts)) / float64(len(hugeUsers)),
			TopCategories:       []string{"cat0", "cat1", "cat2", "topic0", "topic1"},
			MonthlyStats:        monthlyStats,
		},
		Metadata: Metadata{
			Version:     "1.0.0",
			Generated:   time.Now().Format(time.RFC3339),
			UserCount:   len(hugeUsers),
			PostCount:   len(hugePosts),
			Description: "Large dataset for performance testing",
		},
	}
	if err := writeGlintFile(fmt.Sprintf("%s/huge-go.glint", outputDir), hugeData); err != nil {
		log.Printf("Error writing huge dataset: %v", err)
	}
	if err := writeJSONFile(fmt.Sprintf("%s/huge-go.json", outputDir), hugeData); err != nil {
		log.Printf("Error writing huge JSON: %v", err)
	}
	
	fmt.Println("\nTest data generated successfully!")
}