GOLANG_VERSION ?= 1.13

.PHONY: all clean

build/%: lambdas/%
	docker run -t --rm \
	  -w /src/ \
	  -v $(PWD):/src/ \
	  --entrypoint bash \
	  golang:$(GOLANG_VERSION)-buster -c 'go build -o /src/$@ ./$<'

%.zip: build/%
	zip -j $@ $<

all: sign-package.zip create-repo-metadata.zip sign-repo-metadata.zip

clean:
	rm -rf build *.zip
