package mygit

func sanitizeURL(url string) string {
	if url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	return url
}
