FROM --platform=$BUILDPLATFORM alpine AS terraform-downloader
ARG TARGETARCH

RUN apk add --no-cache curl unzip
RUN curl -fsSL -o terraform.zip https://releases.hashicorp.com/terraform/1.14.7/terraform_1.14.7_linux_${TARGETARCH}.zip \
    && unzip terraform.zip \
    && mv terraform /usr/local/bin/terraform

FROM alpine
COPY kubara /kubara
COPY --from=terraform-downloader /usr/local/bin/terraform /usr/local/bin/terraform

RUN apk add --no-cache helm kubectl

ENTRYPOINT ["/kubara", "--kubeconfig", "/kubeconfig"]
