FROM scratch
ENTRYPOINT ["/gohome", "--auto=false"]
CMD ["--bind=:80"]
EXPOSE 80
VOLUME /.cache
LABEL org.opencontainers.image.source https://github.com/ebnull/gohome
COPY gohome /
