# Build the mattermost oncall handler
ARG DOCKER_BUILD_IMAGE=golang:1.17.0
ARG DOCKER_BASE_IMAGE=alpine:3.14

FROM ${DOCKER_BUILD_IMAGE} AS build
WORKDIR /mattermost-oncall-handler/
COPY . /mattermost-oncall-handler/
RUN apt-get update -yq && apt-get install -yq unzip
RUN make build


# Final Image
FROM ${DOCKER_BASE_IMAGE}
LABEL name="Mattermost Oncall Handler" \
  maintainer="cloud-team@mattermost.com" \
  vendor="Mattermost" \
  distribution-scope="public" \
  url="https://mattermost.com" \
  io.k8s.description="Mattermost Oncall Handler is responsible for handling oncall and support Mattermost group updates" \
  io.k8s.display-name="Mattermost Oncall Handler"

ENV CLOUD=/mattermost-oncall-handler/main \
    USER_UID=10001 \
    USER_NAME=cloud
    
RUN  apk update && apk add libc6-compat && apk add ca-certificates && apk --no-cache add tzdata
COPY --from=build /mattermost-oncall-handler/build/_output/bin/main /mattermost-oncall-handler/main
COPY --from=build /mattermost-oncall-handler/build/bin /usr/local/bin

RUN  /usr/local/bin/user_setup
WORKDIR /mattermost-oncall-handler/

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
