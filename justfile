#!/usr/bin/env just --justfile

help:
  @just --list

format:
    templ fmt .
    gofmt -w .

prebuild:
    go mod tidy
    pnpm install
    templ generate
    pnpm exec tailwindcss -i ./internal/web/view/styles.css -o ./internal/web/view/styles.min.css --minify
    pnpm exec tsgo --outDir ./internal/web/view

dev:
    air

bulid:
    @just prebuild
    go build -o viewre

run:
    @just prebuild
    go run main.go
