package main
import (
  "encoding/json"
  "fmt"
  "io"
  http "github.com/bogdanfinn/fhttp"
  tls_client "github.com/bogdanfinn/tls-client"
  "github.com/bogdanfinn/tls-client/profiles"
)
func main() {
  client,_ := tls_client.NewHttpClient(tls_client.NewNoopLogger(),
    tls_client.WithClientProfile(profiles.Chrome_146),
    tls_client.WithTimeoutSeconds(20),
    tls_client.WithDisableHttp3(),
    tls_client.WithRandomTLSExtensionOrder(),
  )
  req,_ := http.NewRequest("GET","https://tls.peet.ws/api/all",nil)
  req.Header.Set("user-agent","Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")
  resp,err := client.Do(req); if err!=nil { panic(err) }
  b,_ := io.ReadAll(resp.Body); resp.Body.Close()
  var m map[string]any; json.Unmarshal(b,&m)
  tls := m["tls"].(map[string]any)
  h2 := m["http2"].(map[string]any)
  fmt.Println("ja4", tls["ja4"])
  fmt.Println("ja3_hash", tls["ja3_hash"])
  fmt.Println("peetprint_hash", tls["peetprint_hash"])
  fmt.Println("h2_akamai", h2["akamai_fingerprint"])
  fmt.Println("h2_hash", h2["akamai_fingerprint_hash"])
}
