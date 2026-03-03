import { ApolloClient, InMemoryCache, HttpLink, split } from '@apollo/client'
import { getMainDefinition } from '@apollo/client/utilities'
import { Observable } from '@apollo/client/core'
import { print } from 'graphql'
import { createClient } from 'graphql-sse'
import type { FetchResult, Operation } from '@apollo/client'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const httpLink = new HttpLink({
  uri: `${API_URL}/graphql`,
})

const sseClient = createClient({
  url: `${API_URL}/graphql`,
})

class SSELink {
  request(operation: Operation): Observable<FetchResult> {
    return new Observable((sink) => {
      const dispose = sseClient.subscribe(
        {
          query: print(operation.query),
          variables: operation.variables,
          operationName: operation.operationName,
        },
        {
          next: (data) => sink.next(data as FetchResult),
          error: (err) => {
            if (err instanceof Error) {
              sink.error(err)
            } else if (err instanceof CloseEvent) {
              sink.error(new Error(`SSE connection closed: ${err.reason}`))
            } else {
              sink.error(new Error(`SSE error`))
            }
          },
          complete: () => sink.complete(),
        },
      )
      return () => dispose()
    })
  }
}

const sseLink = new SSELink()

const splitLink = split(
  ({ query }) => {
    const definition = getMainDefinition(query)
    return (
      definition.kind === 'OperationDefinition' &&
      definition.operation === 'subscription'
    )
  },
  sseLink as never,
  httpLink,
)

export const client = new ApolloClient({
  link: splitLink,
  cache: new InMemoryCache(),
})
