ARG ASTIANGO_TAG=latest

FROM goastian/astiango-hub-backend:${ASTIANGO_TAG} AS backend-build

FROM goastian/astiango-hub-frontend:${ASTIANGO_TAG} AS frontend-build

FROM goastian/astiango-hub-base:${ASTIANGO_TAG}

# Copy files
COPY --from=backend-build /go/bin/astiango-hub-server /usr/local/bin/astiango-hub-server
COPY --from=frontend-build /app/dist /app/dist
COPY ./backend/conf /app/backend/conf
COPY ./docker/nginx/astiango-hub.conf /etc/nginx/conf.d
COPY ./docker/bin/docker-init.sh /app/bin/docker-init.sh
COPY ./docker/bin/health-check.sh /app/bin/health-check.sh

# Start backend
CMD ["/bin/bash", "/app/bin/docker-init.sh"]

# Frontend port
EXPOSE 8080

# Healthcheck for backend
HEALTHCHECK --interval=1m --timeout=3s \
  CMD bash /app/bin/health-check.sh || exit 1
