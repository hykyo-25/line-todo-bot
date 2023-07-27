package schedulerbot

import (
	"context"
	"log"
	"net/http"
	"fmt"
	"strings"
	"sort"
	"github.com/hykyo-25/scheduler-bot/cloudsql"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"github.com/line/line-bot-sdk-go/linebot"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	tasks "google.golang.org/api/tasks/v1"
)

func AllEcho(w http.ResponseWriter, r *http.Request) {
	user_token_list, err := cloudsql.GetAllUserTokens()
	if err != nil {
        log.Print(err)
        return
    }

	bot, err := createBot()
    if err != nil {
        log.Print(err)
        return
    }

	for i, user := range user_token_list {
		user_id := user.UserID
        refreshToken := user.RefreshToken
		return_items, err := getTaskList(refreshToken)
		if err != nil {
			log.Fatalf("Failed to get task: %v", err)
			return
		}
		msg := "タスク一覧\n"
		for _, item := range return_items {
			msg += strings.Replace(item["due"][:10], "-", "/", -1) + ": " + item["title"] + "\n"
		}
		if err := sendMessage(bot, user_id, msg); err != nil {
			log.Fatalf("Failed to send message")
			return
		}
		log.Printf("all echo: %d", i)
	}
} 

func sendMessage(bot *linebot.Client, user_id, message string) error {
    _, err := bot.PushMessage(user_id, linebot.NewTextMessage(message)).Do()
    if err != nil {
        log.Print(err)
        return err
    }

    return nil
}

func createBot() (*linebot.Client, error) {
    CHANNEL_SECRET, err := accessSecretVersion("CHANNEL_SECRET", "1")
    if err != nil {
        log.Print(err)
        return nil, err
    }
    CHANNEL_TOKEN, err := accessSecretVersion("CHANNEL_TOKEN", "1")
    if err != nil {
        log.Print(err)
        return nil, err
    }

    bot, err := linebot.New(CHANNEL_SECRET, CHANNEL_TOKEN)
    if err != nil {
        log.Print(err)
        return nil, err
    }

    return bot, nil
}

func getTaskList (refreshToken string) ([]map[string]string, error) {
    ctx := context.Background()
    config := loadOAuthConfig()

    // Construct a new token source using the provided tokens and config
	token := &oauth2.Token{
        AccessToken: "",
        RefreshToken: refreshToken,
    }
    ts := config.TokenSource(ctx, token)

    // Construct a new oauth client
    client := oauth2.NewClient(ctx, ts)

    service, err := tasks.New(client)
    if err != nil {
        log.Printf("Unable to create tasks service: %v", err)
        return nil, err
    }

    taskList, err := service.Tasklists.List().MaxResults(10).Do()
    if err != nil {
        log.Printf("Unable to retrieve task list: %v", err)
        return nil, err
    }

    task_id := taskList.Items[0].Id

    tasks, err := getTasksFromTaskList(service, task_id)
    if err != nil {
        log.Printf("Unable to retrieve tasks from task list: %v", err)
        return nil, err
    }

    return_items := parceTaskItems(tasks)

    return return_items, nil
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

func loadOAuthConfig() *oauth2.Config {
	// Load OAuth config from credentials file
    OAUTH_CREDS, err := accessSecretVersion("OAUTH_CREDS", "1")
    if err != nil {
        log.Print(err)
    }
	b := []byte(OAUTH_CREDS)

	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/tasks.readonly")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	return config
}

func accessSecretVersion(secretId string, version string) (string, error) {
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to create secretmanager client: %v", err)
	}

	// Build the request.
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/%s", "hayakawa-selenium", secretId, version),
	}

	// Call the API.
	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		log.Fatalf("failed to access secret version: %v", err)
	}

	return string(result.Payload.Data), err
}