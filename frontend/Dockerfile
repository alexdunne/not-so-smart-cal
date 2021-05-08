FROM node:14.16-alpine

WORKDIR /app
EXPOSE 3000
CMD ["npm", "run", "dev"]

COPY package* ./
# exclude package-lock.json to minimize updates 
RUN npm install --no-package-lock
COPY . .