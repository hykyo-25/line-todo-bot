package cloudsql

import (
        "context"
	"database/sql"
	"fmt"
	"log"

        _ "github.com/go-sql-driver/mysql"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
)

type UserToken struct {
        UserID string
        AccessToken string
        RefreshToken string
}

func ConnectUnixSocket() (*sql.DB, error) {
        var (
                dbUser         = accessSecretVersion("DB_USER", "1")              // e.g. 'my-db-user'
                dbPwd          = accessSecretVersion("DB_PASS", "1")              // e.g. 'my-db-password'
                dbName         = accessSecretVersion("DB_NAME", "1")              // e.g. 'my-database'
                unixSocketPath = accessSecretVersion("INSTANCE_UNIX_SOCKET", "1") // e.g. '/cloudsql/project:region:instance'
        )

        dbURI := fmt.Sprintf("%s:%s@unix(%s)/%s?parseTime=true",
                dbUser, dbPwd, unixSocketPath, dbName)

        // dbPool is the pool of database connections.
        dbPool, err := sql.Open("mysql", dbURI)
        if err != nil {
                return nil, fmt.Errorf("sql.Open: %w", err)
        }

        // ...

        return dbPool, nil
}

func GetAllUserTokens() ([]UserToken, error) {
        db, err := ConnectUnixSocket()
        if err != nil {
            return nil, fmt.Errorf("ConnectUnixSocket: %w", err)
        }
        defer db.Close()
    
        rows, err := db.Query("SELECT * FROM tokens")
        if err != nil {
            return nil, fmt.Errorf("db.Query: %w", err)
        }
        defer rows.Close()
    
        var users []UserToken
        for rows.Next() {
            var u UserToken
            if err := rows.Scan(&u.UserID, &u.AccessToken, &u.RefreshToken); err != nil {  // adjust this line based on your table schema
                return nil, fmt.Errorf("rows.Scan: %w", err)
            }
            users = append(users, u)
        }
    
        if err := rows.Err(); err != nil {
            return nil, fmt.Errorf("rows.Err: %w", err)
        }
    
        return users, nil
    }  

func SaveTokens(userID, accessToken, refreshToken string) error {
        db, err := ConnectUnixSocket()
        if err != nil {
                log.Fatalf("Failed to connect to database: %v", err)
        }
        defer db.Close()
        query := `INSERT INTO tokens (user_id, access_token, refresh_token) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE access_token = ?, refresh_token = ?`
    
        _, err = db.Exec(query, userID, accessToken, refreshToken, accessToken, refreshToken)
        if err != nil {
            log.Printf("Failed to save tokens for user %s: %v", userID, err)
            return fmt.Errorf("could not save tokens: %w", err)
        }
    
        return nil
    }

func GetTokens(userID string) (string, string, error) {
        db, err := ConnectUnixSocket()
        if err != nil {
                log.Fatalf("Failed to connect to database: %v", err)
        }
        defer db.Close()

        var accessToken, refreshToken string
        query := `SELECT access_token, refresh_token FROM tokens WHERE user_id = ?`
    
        err = db.QueryRow(query, userID).Scan(&accessToken, &refreshToken)
        if err != nil {
            log.Printf("Failed to get tokens for user %s: %v", userID, err)
            return "", "", fmt.Errorf("could not get tokens: %w", err)
        }
    
        return accessToken, refreshToken, nil
    }

func accessSecretVersion(secretId string, version string) string {
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

	return string(result.Payload.Data)
}