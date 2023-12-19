package send

import (
  "bytes"
  "fmt"
  "io/ioutil"
  "net/http"
  "time"

  "github.com/golang/snappy"
  "github.com/prometheus/prometheus/prompb"
)

// Send function sends the given metrics to the given endpoint
func Send(endpoint string, ts []prompb.TimeSeries, debug bool) {
    // Create a WriteRequest
    req := &prompb.WriteRequest{
        Timeseries: ts,
    }

    // Marshal the WriteRequest to a byte slice
    data, err := req.Marshal()
    if err != nil {
        fmt.Printf("Could not marshal the WriteRequest: %v\n", err)
        return
    }

    // Compress the byte slice
    compressed := snappy.Encode(nil, data)

    // Create a HTTP request
    httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(compressed))
    if err != nil {
        fmt.Printf("Could not create HTTP request: %v\n", err)
        return
    }

    // Set headers
    httpReq.Header.Set("Content-Encoding", "snappy")
    httpReq.Header.Set("Content-Type", "application/x-protobuf")
    httpReq.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

    // Send the request
    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(httpReq)
    if err != nil {
        fmt.Printf("Could not send HTTP request: %v\n", err)
        return
    }

    // Print debug info, if required
    if debug {
        body, _ := ioutil.ReadAll(resp.Body)
        fmt.Printf("Debug: Send response: %s\n", string(body))
    }

    // Close the response body
    resp.Body.Close()

    // Check the response code, now for both 200 and 204
    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
        fmt.Printf("Received a non-200/204 response code: %d\n", resp.StatusCode)
    }
}
