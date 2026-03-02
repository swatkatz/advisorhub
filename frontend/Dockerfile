FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL
RUN npm run build

FROM node:22-alpine
WORKDIR /app
RUN npm install -g serve@14
COPY --from=builder /app/dist ./dist
ENV PORT=8080
EXPOSE $PORT
CMD ["sh", "-c", "serve -s dist -l $PORT"]
