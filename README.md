<div align="center">

[![deepl-go][repo_logo_img]][repo_url]

# deepl-go
[![Go Report Card][report_card_badge]][report_card_url]
[![Lisence: MIT][license_badge]][license_url]

`deepl-go` is an unofficial Go client library for the [DeepL translation API][deepl_api_docs]. It enables developers to integrate DeepL's powerful translation services into their Go applications effortlessly.
</div>

---

## 🔑 Obtaining a DeepL API Key

To use `deepl-go`, you need a DeepL API key:

1. Create a [DeepL account][deepl_signup_url].
2. Subscribe to a plan that provides API access (either Free or Pro).
3. Once subscribed, you will receive an API key, which you should keep private and use in your application as the authorization token.

**Note:** If your API key ends with `:fx`, it is recognized as a Free API key and the client will automatically use the free API endpoint.
A DeepL API Free account allows you to use up to 500,000 characters per month at no cost.

---

## 📋 Requirements

- Go 1.20 or newer

---

## ⚙ Installation

You can install deepl-go using `go get`:

```bash
go get github.com/lkretschmer/deepl-go
```

---

## 🚀 Usage

### Creating a Client
Create a new DeepL client by providing your API key:
```go
client := deepl.NewClient("your_api_key_here")
```
### Translating Text
Translate a single string to a target language:
```go
translation, err := client.TranslateText("Hello, world!", "DE")
if err != nil {
    log.Fatalf("translation error: %v", err)
}
fmt.Println("Translated text:", translation.Text)
```

---

## 📄 License
This project is licensed under the MIT License.
See the [LICENSE][license_url] file for details.

---

## ⚠️ Disclaimer
This client library is not an official DeepL product. For official documentation and terms, please visit [DeepL API][deepl_api_docs].



[repo_url]: https://github.com/lkretschmer/deepl-go
[repo_logo_img]: https://raw.githubusercontent.com/lkretschmer/deepl-go/refs/heads/master/.github/logo.svg
[report_card_badge]: https://goreportcard.com/badge/github.com/lkretschmer/deepl-go
[report_card_url]: https://goreportcard.com/report/github.com/lkretschmer/deepl-go
[license_badge]: https://img.shields.io/badge/license-MIT-blueviolet.svg
[license_url]: https://github.com/lkretschmer/deepl-go/blob/main/LICENSE
[deepl_api_docs]: https://developers.deepl.com/docs
[deepl_signup_url]: https://www.deepl.com/en/signup