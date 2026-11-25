FROM node:22-alpine
WORKDIR /app
COPY package.json ./
RUN npm install --omit=dev
COPY server.ts ./
ENV NODE_ENV=production
CMD ["node", "server.ts"]