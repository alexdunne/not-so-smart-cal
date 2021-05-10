import { ApolloServer } from "apollo-server";
import { makeSchema } from "nexus";
import path from 'path'

import * as types from './types'

const schema = makeSchema({
  types,
  outputs: {
    schema: path.join(__dirname, "generated", "schema.graphql"),
    typegen: path.join(__dirname, "generated", "nexus.d.ts"),
  },
})

const server = new ApolloServer({
  schema,
})


const port = process.env.PORT || 4000

server.listen({ port }, () => console.log(`🚀 GraphQL API ready at http://localhost:${port}${server.graphqlPath}`))