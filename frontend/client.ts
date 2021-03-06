import { ApolloClient, InMemoryCache } from "@apollo/client";
import getConfig from "next/config";

const { publicRuntimeConfig } = getConfig();

export const client = new ApolloClient({
  uri: publicRuntimeConfig.API_URL,
  cache: new InMemoryCache(),
});
