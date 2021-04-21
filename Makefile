gen_easyjson:
	rm -f requests_easyjson.go responses_easyjson.go
	easyjson -all requests.go
	easyjson -all responses.go