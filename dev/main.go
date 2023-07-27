package main

import (
	"context"
	// "encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sort"

	tasks "google.golang.org/api/tasks/v1"
)

var (
	config *oauth2.Config
	ctx    = context.Background()
)

func referArray4Int(ary *[4]int) {
    fmt.Printf("Pointer: %p , Value: %v\n", &ary, ary)
	fmt.Printf("Pointer: %p , Value: %v\n", ary, ary)
}

func main() {
	// Load OAuth config from credentials file

	arr1 := [4]int{1, 2, 3, 4}
	arr2 := [4]int{1, 2, 3, 4}
	fmt.Printf("Pointer: %p , %v\n", &arr1, arr1 == arr2)
	referArray4Int(&arr1)

	slc1 := []byte{0, 1, 2, 3, 4}
    fmt.Printf("Type: %[1]T , Value: %[1]v\n", slc1)

	return

	b, err := ioutil.ReadFile("credentials-desktop.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err = google.ConfigFromJSON(b, "https://www.googleapis.com/auth/tasks.readonly")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	// Start the server
	r := gin.Default()

	r.GET("/auth", handleAuth)
	r.GET("/auth/callback", handleAuthCallback)
	r.GET("/tasks", handleTasks)

	r.Run()
}

func handleAuth(c *gin.Context) {
	url := config.AuthCodeURL("U1919919191", oauth2.AccessTypeOffline)
	c.Redirect(http.StatusFound, url)
}

func handleAuthCallback(c *gin.Context) {
	code := c.Query("code")
	tok, err := config.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Print("Access Token: " + tok.AccessToken)
    log.Print("Refresh Token: " + tok.RefreshToken)

	// Save the token for the subsequent requests
	c.SetCookie("token", tok.AccessToken, 3600, "", "", false, true)

	c.JSON(http.StatusOK, gin.H{"status": "Success"})
}

func handleTasks(c *gin.Context) {
	accessToken, err := c.Cookie("token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	service, err := tasks.New(tc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskList, err := service.Tasklists.List().MaxResults(10).Do()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

        task_id := taskList.Items[0].Id

        tasks, err := getTasksFromTaskList(service, task_id)
        if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

        return_items := parceTaskItems(tasks)

	c.JSON(http.StatusOK, return_items)
}

func parceTaskItems(tasks *tasks.Tasks) []map[string]string {
        task_items := tasks.Items
        return_items := make([]map[string]string, 0)
        for _, item := range task_items {
                fmt.Println(item.Title, item.Due)
                tmp_map := map[string]string{
                        "title" : item.Title,
                        "due": item.Due,
                        "position": item.Position,
                }
                return_items = append(return_items, tmp_map)
        }

		return_items = sortItems(return_items)

		msg := "タスク一覧\n"
		for _, item := range return_items {
			msg += strings.Replace(item["due"][:10], "-", "/", -1) + ": " + item["title"] + "\n"
		}
		fmt.Print(msg)

        return return_items
}

func sortItems(return_items []map[string]string) []map[string]string {
	pairs := make([]struct {
		key string
		value map[string]string
	}, len(return_items))
	for i, m := range return_items {
		pairs[i].key = m["position"]
		pairs[i].value = m
	}

	// Sort the pairs slice.
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})

	// Create a new slice containing the sorted maps.
	sortedData := make([]map[string]string, len(return_items))
	for i, pair := range pairs {
		sortedData[i] = pair.value
	}
	return sortedData
}

func getTasksFromTaskList(service *tasks.Service, taskListId string) (*tasks.Tasks, error) {
	tasks, err := service.Tasks.List(taskListId).Do()
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

