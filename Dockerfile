FROM node:16-alpine3.13 as jsbuilder

WORKDIR /app

COPY ui/webapp/package.json ./package.json

# We want to cache the node_modules directory
RUN npm i

COPY ui/webapp .

RUN npm run build

FROM golang:1.16-alpine3.13 as gobuilder

WORKDIR /opt/sds/app

# Copy only the Go source files over
COPY cmd .
COPY internal .
COPY ui/sds_ui.go ./ui/
COPY vendor .

# Copy the built JS app from the previous stage
COPY --from=jsbuilder /app/dist ./ui/webapp/dist

# Put everything together
RUN make sds

FROM busybox:1.33.1

WORKDIR /opt/sds

# Copy over the built static binary
COPY --from=gobuilder /opt/sds/sigplot_data_service .

EXPOSE 5055

ENTRYPOINT [ "sigplot-data-service" ]

CMD "-h"