package main
import (
  "fmt"
  "io"
  "os"
  http "github.com/bogdanfinn/fhttp"
  tls_client "github.com/bogdanfinn/tls-client"
  "github.com/bogdanfinn/tls-client/profiles"
)
func main() {
  client, _ := tls_client.NewHttpClient(tls_client.NewNoopLogger(), tls_client.WithClientProfile(profiles.Chrome_133), tls_client.WithTimeoutSeconds(20))
  for _, u := range []string{"https://api.ipify.org", "https://ip.hypersolutions.co/ip"} {
    req, _ := http.NewRequest("GET", u, nil)
    req.Header.Set("user-agent", "Mozilla/5.0")
    if u == "https://ip.hypersolutions.co/ip" {
      req.Header.Set("x-api-key", os.Getenv("HYPER_API_KEY"))
    }
    resp, err := client.Do(req)
    if err != nil { fmt.Println(u, err); continue }
    b, _ := io.ReadAll(resp.Body); resp.Body.Close()
    fmt.Println(u, resp.StatusCode, string(b))
  }
}
