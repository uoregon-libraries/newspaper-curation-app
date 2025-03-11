# Acceptance Tests

This is our Go-powered acceptance test suite. Here you'll find tests for
validating the full NCA + Agent + ONI setup. This is a standalone project: it
doesn't use NCA or ONI code directly, and can be run against any NCA (in debug
mode) and ONI setup you might have.

To run this test you need a browser that supports the Chrome DevTools Protocol,
but a headless chrome is usually the easiest approach. A `compose.yml` file is
provided which can stand up a headless browser, bound to your local network in
order to connect to whatever local instances of NCA and ONI you have running.

Once you have a CDP-enabled browser running (the compose setup or otherwise),
you simply have to set up some environment variables and run `go test` to
invoke the test suite. An example, if you are developing on NCA in "hybrid"
mode, might look like this:

```bash
export NCA_URL=http://localhost:3333
export ONI_PROD_URL=http://localhost:8080
export ONI_STAGING_URL=http://localhost:8082
export HEADLESS_URL=http://localhost:3000

go test
```
