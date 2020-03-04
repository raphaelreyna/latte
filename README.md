# LaTTe
Generate PDFs using LaTeX templates and JSON.

## How to use LaTTe
LaTTe starts an HTTP server and listens on port 27182 by default at the root endpoint `/`.
Example POST request JSON body:
```
{
  "template": <BASE64 ENCODED .tex FILE>,
  "resources": {
    "image1.png": <BASE64 ENCODED RESOURCE FILE>
   },
   "details": {
    "CustomerName": "John Doe",
    "Phone": "555-555-5555"
   }
}
```
The template `.tex` file should be a template that follows [Go's templating syntax](https://golang.org/pkg/text/template/).

LaTTe currenlty only accepts using `#!` and `!#` as the left and right delimeters (respectively) in the `.tex` template file.

## Environment Variables
### `LATTE_PORT`
The port that LaTTe will bind to. The default value is 27182.
### `LATTE_ROOT`
The directory that LaTTe will use to store all of its files. The default value is the users cache directory.
### `LATTE_DB_HOST`
The address where LaTTe can reach its database (assuming LaTTe was compiled with database support).
### `LATTE_DB_PORT`
The the port that LaTTe will use when connecting to its database (assuming LaTTe was compiled with database support).
### `LATTE_DB_USERNAME`
The username that LaTTe will use to connect to its database (assuming LaTTe was compiled with database support).
### `LATTE_DB_PASSWORD`
The password that LaTTe will use to connect to its database (assuming LaTTe was compiled with database support).
### `LATTE_DB_SSL`
Dictates if the database that LaTTe will use is using SSL; acceptable values are `required` and `disable` (assuming LaTTe was compiled with database support).


## Image size
LaTTe relies on [pdflatex](https://www.tug.org/applications/pdftex/) in order to actually create the PDF files.
Unfortunately, this means that image sizes can be rather large (a full texlive installation is around 4GB).
The build script in the `build` directory makes it easy to create custom sized images of LaTTe to fit your needs.

## Contributing
Contributions are welcome!
### Adding databases / persistent store drivers
LaTTe can easily be extended to support using various databases and other storage solutions.
To have LaTTe use your persistnent storage solution of choice, simply create a struct that satisfies the `DB` interface:
```
type DB interface {
	// Store should be capable of storing a given []byte or contents of an io.ReadCloser
	Store(ctx context.Context, uid string, i interface{}) error
	// Fetch should return either a []byte, or io.ReadCloser.
	// If the requested resource could not be found, error should be of type NotFoundError
	Fetch(ctx context.Context, uid string) (interface{}, error)
	// Ping should check if the databases is reachable.
  // If it is, the return error should be nil and non-nil otherwise.
	Ping(ctx context.Context) error
}
```

## Roadmap
- :heavy_check_mark: <s>Registering templates and resources.</s>
- Add support for AWS S3, <s>PostrgeSQL</s>, and possibly other forms of persistent storage.
- CLI tool.
- Add support for building PDFs from multiple LaTeX files.
- Whatever else comes up
