# CopyPaste

A self-hosted solution to share text and files through a web-based interface without hassles

## Installation

> [!WARNING]  
> CopyPaste uses http. Be sure to use to secure your connection if
> you intend to use CopyPaste outside your local network

Just clone the repo and start the container

```sh
cd ./copypaste
docker compose up -d --build
```

## Functions

### Clipboard

Put any text you want and it will be displayed to be copied on other devices

### Files

Upload the files you want to share with other devices

## TODO

- [x] ~Implement files management~
- [ ] Add icons based on filetypes
- [ ] Auto updating clipboard
- [x] ~Improve the UI~
- [x] ~Reactive UI~
- [ ] Multistage docker build with css generation included
- [ ] Write tests
