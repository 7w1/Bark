# BaaS (Bark as a Service)

A low-latency Go API for generating and detecting dog vocalizations. It runs as a statically linked binary in a small Docker container.

Website: [bark.plz.pet](https://bark.plz.pet)

## API Endpoints

### 1. Generate

`GET /bark`

Generates a vocalization using procedural phonetic combinations.

**Example response:**

```json
{
  "generation_time": "14.625µs",
  "bark": "woooof",
  "family": "woof",
  "language": "en"
}
```

### 2. Detect

`GET /isbark?text={input}`

Checks whether the entire input is a dog sound, including stretched spellings like `wooooffff`.

**Example response:**

```json
{
  "detection_time": "18.210µs",
  "input": "woof",
  "is_bark": true,
  "bark_percentage": "100.0%",
  "squeezed_form": "wof",
  "family": "woof",
  "match_type": "exact",
  "language": "en"
}
```

### 3. Detect bark-like text

`GET /barklike?text={input}`

Recognizes barks and other dog sounds in short messages.

### 4. Bark 8-ball

`GET /askbark?text={question}`

Returns a deterministic answer in English.

### Languages

Add `lang=fr` (or another supported tag) to `/bark`, `/isbark`, or `/barklike`. These endpoints default to English. Detection also accepts `lang=auto` to find the language. `GET /languages` lists the supported tags.

## Configuration

The API reads `config.json` from the root directory. If missing, it uses default values. For example:

```json
{
  "port": "8000",
  "generate_endpoint": "/bark",
  "detect_endpoint": "/isbark"
}
```

## Local Development

Requires Go 1.26.3 or later.

```bash
go run .
go test ./...
```
