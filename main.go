package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type MovieData struct {
	Data struct {
		Items []struct {
			Formats []struct {
				Sessions []struct {
					ID int `json:"id"`
				} `json:"sessions"`
			} `json:"formats"`
		} `json:"items"`
	} `json:"data"`
}

type OrderResponse struct {
	Data struct {
		OrderKey string `json:"orderKey"`
	} `json:"data"`
}

type SeatsResponse struct {
	Data struct {
		Seats []struct {
			Rows []struct {
				Name  string `json:"name"`
				Seats []struct {
					ID       string `json:"id"`
					Position struct {
						Area   int `json:"area"`
						Row    int `json:"row"`
						Column int `json:"column"`
					} `json:"position"`
				} `json:"seats"`
			} `json:"rows"`
		} `json:"seats"`
	} `json:"data"`
}

func main() {
	userInput := "https://www.karofilm.ru/film/13169?date=2024-08-01"
	parts := strings.Split(userInput, "/")
	movieID := strings.Split(parts[len(parts)-1], "?")[0]

	movieDataURL := fmt.Sprintf("https://api.karofilm.ru/movie-schedule?city_id=13&movie_id=%s", movieID)
	movieDataResp, err := http.Get(movieDataURL)
	if err != nil {
		panic(err)
	}

	var movieData MovieData
	if err := json.NewDecoder(movieDataResp.Body).Decode(&movieData); err != nil {
		panic(err)
	}
	movieDataResp.Body.Close()

	for _, data := range movieData.Data.Items {
		for _, format := range data.Formats {
			for _, session := range format.Sessions {
				sessionID := session.ID

				orderPayload := url.Values{}
				orderPayload.Set("stream_key", "2dc28ac0918441379a5a42dfaca38082")
				orderPayload.Set("start", "film")
				orderPayload.Set("delivery", "2")
				orderPayload.Set("session_id", strconv.Itoa(sessionID))

				orderHeader := map[string]string{
					"origin":  "https://www.karofilm.ru",
					"referer": "https://www.karofilm.ru/",
				}

				orderResp, err := makePostRequest("https://api.karofilm.ru/v3/order/create", orderPayload, orderHeader)
				if err != nil {
					panic(err)
				}

				var order OrderResponse
				if err := json.NewDecoder(orderResp.Body).Decode(&order); err != nil {
					panic(err)
				}
				orderResp.Body.Close()

				orderKey := order.Data.OrderKey

				seatsResp, err := http.Get(fmt.Sprintf("https://api.karofilm.ru/v3/session/seats/%s", orderKey))
				if err != nil {
					panic(err)
				}

				var seats SeatsResponse
				if err := json.NewDecoder(seatsResp.Body).Decode(&seats); err != nil {
					panic(err)
				}
				seatsResp.Body.Close()

				for _, seat := range seats.Data.Seats {
					for _, row := range seat.Rows {
						for _, s := range row.Seats {
							confirmPayload := map[string]interface{}{
								"tickets": []map[string]interface{}{
									{
										"details": map[string]interface{}{
											"area_code":            "0000000001",
											"type_code":            "0041",
											"recognition_id":       0,
											"redemption_ticket":    false,
											"price":                510,
											"bonus_price":          0,
											"title":                "Стандартный",
											"loyalty_members_only": false,
											"max_tickets":          3000,
										},
										"seats": []map[string]interface{}{
											{
												"area_code":    "0000000001",
												"area_number":  s.Position.Area,
												"row":          row.Name,
												"row_index":    s.Position.Row + 1,
												"column":       s.ID,
												"column_index": s.Position.Column + 1,
											},
										},
									},
								},
								"products": []interface{}{},
							}

							confirmPayloadBytes, err := json.Marshal(confirmPayload)
							if err != nil {
								panic(err)
							}

							confirmHeader := map[string]string{
								"accept":       "application/json,text/plain,*/*",
								"origin":       "https://www.karofilm.ru",
								"referer":      "https://www.karofilm.ru/",
								"content-type": "application/json",
							}

							confirmResp, err := makePostRequestWithJSON(fmt.Sprintf("https://api.karofilm.ru/v3/order/update/%s", orderKey), confirmPayloadBytes, confirmHeader)
							if err != nil {
								panic(err)
							}

							var data map[string]interface{}
							if err := json.NewDecoder(confirmResp.Body).Decode(&data); err != nil {
								panic(err)
							}
							confirmResp.Body.Close()

							confirmRespGet, err := http.Get(fmt.Sprintf("https://api.karofilm.ru/v3/order/info/%s", orderKey))
							if err != nil {
								panic(err)
							}
							confirmRespGet.Body.Close()

							fmt.Println(data)
						}
					}
				}
			}
		}
	}
}

func makePostRequest(url string, data url.Values, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	return client.Do(req)
}

func makePostRequestWithJSON(url string, data []byte, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	return client.Do(req)
}
