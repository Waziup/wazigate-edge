# Waziup Edge Server *for Waziup Gateway*

The edge server provides basic endpoints do retrieve, upload and change device, sensor and actuator data.

You can use REST and MQTT on all endpoints.

![Waziup Structure](./assets/waziup_structure.svg)


# Usage

This project is part of the [Waziup Open Innovation Platform](https://www.waziup.eu/). In most cases you do not want to use this repository without the
waziup platform, so have a look the [**Waziup Gateway**](https://github.com/Waziup/waziup-gateway) at [github.com/Waziup/waziup-gateway](https://github.com/Waziup/waziup-gateway).


# Development

## with go (golang) from source

You can compile this project from source with golang and git.
Grab yourself the go language from [golang.org](https://golang.org/) and the
git command line tools with `apt-get git` or from [git-scm.com/download](https://git-scm.com/download).

Now build the waziup-edge executable:

```bash
git clone https://github.com/Waziup/waziup-edge.git
cd waziup-edge
go build .
```

And run the waziup-edge server with:


```bash
waziup-edge
```

## with docker

If you like to use docker you can use the public waziup docker containers at [the Docker Hub](https://hub.docker.com/u/waziup/).
For development you can build this repo on your own using:

```bash
git clone https://github.com/Waziup/waziup-edge.git
cd waziup-edge
docker build --tag=waziup-edge .
docker run -p 4000:80 waziup-edge
```
