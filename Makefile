.PHONY: all clean

build/%: lambdas/%
	docker run --rm \
	  -w /usr/src/rpm-lambdas \
	  -v $(PWD):/usr/src/rpm-lambdas \
	  golang:1 go build -o ./$@ ./$<

%.zip: build/%
	zip -j $@ $<

all: sign-package.zip create-repo-metadata.zip sign-repo-metadata.zip

clean:
	rm -rf build *.zip
