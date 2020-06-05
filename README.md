# LaTTe
![Docker Pulls](https://img.shields.io/docker/pulls/raphaelreyna/latte)

Generate PDFs using LaTeX templates and JSON.

[Try out the demo!](https://raphaelreyna.works/latte)

[Find LaTTe on Docker Hub](https://hub.docker.com/r/raphaelreyna/latte)

## Table of Contents
* [About](#toc-about)
* [Obtaining LaTTe](#toc-obtaining)
* [Running & Using LaTTe](#toc-running-latte)
	* [HTTP Service](#toc-http-service)
		* [Environment Variables](#toc-env-vars)
		* [Registering Files](#toc-registering-files)
		* [Generating PDFs](#toc-service-generating-pdfs)
			* [Example](#toc-example-1)
	* [CLI](#toc-cli)
* [Extending LaTTe](#toc-extending)
* [Docker Images](#toc-docker)
	* [Tags](#toc-tags)
	* [Image Size](#toc-image-size)
	* [Building Custom Images](#toc-building-custom-images)
* [Contributing](#toc-contributing)
* [Roadmap](#toc-roadmap)

<a name="toc-about"></a>
## About
LaTTe helps you generate professional looking PDFs by using your .tex files as templates and filling in the details with JSON.
Under the hood, it uses [pdfTeX / pdfLaTeX](https://tug.org/applications/pdftex) to create a PDF from a .tex file that has been filled in using [Go's templating package](https://golang.org/pkg/text/template/).

LaTTe has two modes of operation:
* As a service over HTTP, while also offering some degree of templating/pre-processing, caching and persistence. [More info on running LaTTe as an HTTP service.](#toc-http-service)
* As a cli tool to create PDFs using TeX/LaTeX and JSON files quickly and easily. [More info on using LaTTe's cli.](#toc-cli)

<a name="toc-obtaining"></a>
## Obtaining LaTTe
You can download the source code for LaTTe by running `git clone github.com/raphaelreyna/latte` in your terminal.
LaTTe can then be easily compiled by running `go build ./cmd/latte`. 
If you wish to build LaTTe with support for PostreSQL, simply run `go build -tags postgresql` instead. [More info on persistent storage support](#toc-extending)

LaTTe is also available via several docker images; running `docker run --rm -d -p 27182:27182 raphaelreyna/latte` will leave you with a basic version of LaTTe running as a an HTTP service.
[More info on Docker images](#toc-docker)

<a name="toc-running-latte"></a>
## Running & Using LaTTe

<a name="toc-http-service"></a>
### HTTP Service
LaTTe will run as an HTTP service by default; running `latte` in your terminal will have LaTTe listening for HTTP traffic on port 27182 by default.
All configurations are done through environment variables.

<a name="toc-env-vars"></a>
### Environment Variables
### `PORT`
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
### `LATTE_TMPL_CACHE_SIZE`
How many templates LaTTe will keep cached in memory. (defaults to 15)
### `LATTE_RSC_CACHE_SIZE`
How many resource files will LaTTe will keep cached in memory. (defaults to 15)

<a name="toc-registering-files"></a>
#### Registering a file
Files are registered by sending an HTTP POST request to the endpoint "/register" with a JSON body of the form:
```
{
	"id": "WHATEVER_NAME_YOU_WANT"
	"data": "BASE_64_ENCODED_STRING"
}
```

<a name="toc-service-generating-pdfs"></a>
#### Generating PDFs
LaTTe can genarate PDF's from both registered and unregistered resources, templates and json files (which LaTTe calls 'details'). A resource is any kind of file used in compiling the .tex file into a PDF (e.g. images); a template is any valid .tex file.

A registered file is one that has been stored either to LaTTe's local disk and/or to some database (currently only PostgreSQL is supported). Registered files are referenced by an ID. When generating a PDF, all references to registered files are made in the URL of the request; unregistered files are provided as base 64 encoded strings in a JSON body.

LaTTe also allows for users to define how missing template keys should be handled through the "onMissingKey" field in the requests JSON body or URL. Valid options for "onMissingKey" are: "error" for when LaTTe should return with a "Bad Request" HTTP status, "zero" for when LaTTe should use the field types zero value ("" for strings, false for booleans, 0 for numerics, etc.).


PDF's are generated by sending an HTTP POST request to the endpoint "/generate" with a JSON body of the form (if using unregisted files):
```
{
	"template": "BASE_64_ENCODED_STRING",
	"resources": {
		"FILE_NAME": "BASE_64_ENCODED_STRING"
		},
	"details": { SOME_OBJECT_DESCRIBING_YOUR_SUBSTITUTIONS },
	"delimiters": { "left": "LEFT_DELIMITER", "right": "RIGHT_DELIMITER" },
   	"onMissingKey": "error" | "zero" | "nothing",
}
```
If you wish to also use registered files, you may reference them in the URL:
```
http://localhost:27182/generate?tmpl=TEMPALATE_ID&rsc="RESOURCE_ID&rsc="SOME_OTHER_RESOURCE_ID"&dtls="DETAILS_ID"&onMissingKey="error|zero|nothing"
```
If you provide both a reference to a file and include it in the JSON body, the file you sent in the body will be used.

<a name="toc-example-1"></a>
##### Example: Generating a PDF from unregistered files
Here we demonstrate how to generate a PDF of the Pythagorean theorem, after substituting variables a, b & c for x, y & z respectively.

We create our .tex template file pythagorean_template.tex:
```
\documentclass{article}
\title{LaTTe Sample Document}
\begin{document}
\maketitle
The Pythagorean Theorem: 
$ #!.a!# ^ 2 + #!.b!# ^ 2 = #!.c!# ^ 2 $
\end{document}
```
The template `.tex` file should be a template that follows [Go's templating syntax](https://golang.org/pkg/text/template/).
LaTTe by default uses `#!` and `!#` as the left and right delimeters, respectively, in the `.tex` template file; however [custom delimiters are supported](#toc-service-generating-pdfs). As required by pdfLaTeX, all files must start with the character "\".

We then convert it to base 64:
```
$ cat pythagorean_template.tex | base64
```
which gives the output:
```
XApUaGUgUHl0aGFnb3JlYW4gVGhlb3JlbTogJCMhLmEhIyBeIDIgKyAjIS5iISMgXiAyID0gIyEu
YyEjIF4gMiQKXGJ5ZQo=
```

We then send this to LaTTe:
```
$ curl -X POST -H "Content-Type: application/json" \
-d '{"template":"XApUaGUgUHl0aGFnb3JlYW4gVGhlb3JlbT\
ogJCMhLmEhIyBeIDIgKyAjIS5iISMgXiAyID0gIyEuYyEjIF4gMiQKXGJ5ZQo=", \
"details": { "a": "x", "b": "y", "c": "z" } }' \
--output pythagorean.pdf "http://localhost:27182/generate"
```

which leaves us with the file `pythagorean.pdf` (the image below is a cropped screenshot of `pythagorean.pdf`):
![pythagorean_pdf](/../screenshots/screenshots/screenshot.png?raw=true)

<a name="toc-cli"></a>
### CLI
LaTTe offers a CLI to quickly and easily generate templated PDFs using the files on your computer.
```
Usage: latte [ -t template_tex_file ] [ -d details_json_file ] [ path/to/resources ]

Description: Generate PDFs using TeX / LaTeX templates and JSON.


Flags:
  -t Path to .tex file to be used as the template.

  -d Path to .json file to be used as the details to fill in to the tamplate.
  
Other:
    The final argument is optional and should be a path to resources needed for compilation.
    Resources are any files that are referenced in the .tex file such as image files.
```

<a name="toc-extending"></a>
## Extending LaTTe
### Adding databases / persistent store drivers
LaTTe can easily be extended to support using various databases and other storage solutions.
To have LaTTe use your persistent storage solution of choice, simply create a struct that satisfies the `DB` interface:
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

<a name="toc-docker"></a>
## Docker Images

<a name="toc-tags"></a>
### Image Tags
There are several LaTTe images available to serve a wide range of needs, and they all follow the same tagging convention:
```
	latte:<VERSION>-[pg]-<base/full>
```
where \<VERSION\> denotes the latte version, [pg] if present denotes Postgres support, and \<base/full\> denotes the presence of either [texlive-full](https://packages.ubuntu.com/eoan/texlive-full) or [texlive-base](https://packages.ubuntu.com/eoan/texlive-base).
	
#### Currently Supported Tags
The currently supported tags for LaTTe are:
##### v0.10.2-base
##### v0.10.2-pg-base
##### latest, v0.10.2-full
##### v0.10.2-pg-base

<a name="toc-image-size"></a>
### Image size
LaTTe relies on [pdflatex](https://www.tug.org/applications/pdftex/) in order to actually create the PDF files.
Unfortunately, this means that image sizes can be rather large (a full texlive installation is around 4GB).
The build script in the `build` directory makes it easy to create custom sized images of LaTTe to fit your needs.

<a name="toc-building-custom-images"></a>
### Building Custom Images
LaTTe comes with a build script, build/build.sh, which makes it easy to build LaTTe images with custom Go build flags and tex packages.

```
Usage: build.sh [-h] [-s] [-b build_tag] [-p latex_package] [-t image_tag]
		[-d descriptor] [-H host_name] [-u user_name]

Description: build.sh parametrically builds and tags Docker images for Latte.
             The tag used for the image follows the template bellow:
                 host_name/user_name/image_name:image_tag-descriptor

Flags:
  -b Build tags to be passed to the Go compiler.

  -d Descriptor to be used when tagging the built image.

  -h Show this help text.

  -p LaTeX package to install, must be available in default Ubuntu repos.
     (default: texlive-base)

  -s Skip the image building stage. The generated Dockerfile will be sent to std out.

  -t Tag the Docker image.
     The image will be tagged with the current git tag if this flag is omitted.
     If no git tag is found, we default to using 'latest' as the image tag.

  -u Username to be used when tagging the built image.
     (default: raphaelreyna)

  -H Hostname to be used when tagging the built image.

  -y Do not ask to confirm image tab before building image.
```

<a name="toc-contributing"></a>
## Contributing
Contributions are welcome!

<a name="toc-roadmap"></a>
## Roadmap
- :heavy_check_mark: <s>Registering templates and resources.</s>
- Add support for AWS S3, <s>PostrgeSQL</s>, and possibly other forms of persistent storage.
- :heavy_check_mark: <s>CLI tool</s>.
- Add support for building PDFs from multiple LaTeX files.
- Whatever else comes up
