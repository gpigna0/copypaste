# CopyPaste

A self-hosted solution to share text and files through a web-based interface without hassles

## Installation

> [!WARNING]  
> CopyPaste uses http. Be sure to use to secure your connection if
> you intend to use CopyPaste outside your local network

Just create a directory and put there `compose.yaml` and rename `example.env` to
`.env` for default options, then pull the image and start the container

```bash
mkdir copypaste
cd copypaste

curl -o compose.yaml https://raw.githubusercontent.com/gpigna0/copypaste/main/compose.yaml
curl -o .env https://raw.githubusercontent.com/gpigna0/copypaste/main/example.env

docker compose pull && docker compose up -d

```

## Functions

### Clipboard

Put any text you want and it will be displayed to be copied on other devices

### Files

Upload the files you want to share with other devices

## TODO

- [x] ~Implement files management~
- [ ] Add icons based on filetypes
- [x] ~Auto updating clipboard and files~
- [x] ~Improve the UI~
- [x] ~Reactive UI~
- [ ] Multistage docker build with css generation included
- [ ] Write tests
