import { ApolloServer } from "apollo-server";
import { makeSchema } from "nexus";
import path from "path";
import { context } from "./graphql/context";

import * as types from "./graphql/types";

const schema = makeSchema({
  types,
  outputs: {
    schema: path.join(__dirname, "generated", "schema.graphql"),
    typegen: path.join(__dirname, "generated", "nexus.d.ts"),
  },
  contextType: {
    module: path.join(__dirname, "graphql", "context.ts"),
    export: "Context",
  },
});

const server = new ApolloServer({
  context: context,
  schema,
});

const port = process.env.PORT || 4000;

server.listen({ port }, () => console.log(`🚀 GraphQL API ready at http://localhost:${port}${server.graphqlPath}`));
