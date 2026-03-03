import { gql } from '@apollo/client';
import * as Apollo from '@apollo/client';
export type Maybe<T> = T | null;
export type InputMaybe<T> = Maybe<T>;
export type Exact<T extends { [key: string]: unknown }> = { [K in keyof T]: T[K] };
export type MakeOptional<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]?: Maybe<T[SubKey]> };
export type MakeMaybe<T, K extends keyof T> = Omit<T, K> & { [SubKey in K]: Maybe<T[SubKey]> };
export type MakeEmpty<T extends { [key: string]: unknown }, K extends keyof T> = { [_ in K]?: never };
export type Incremental<T> = T | { [P in keyof T]?: P extends ' $fragmentName' | '__typename' ? T[P] : never };
const defaultOptions = {} as const;
/** All built-in and custom scalars, mapped to their actual values */
export type Scalars = {
  ID: { input: string; output: string; }
  String: { input: string; output: string; }
  Boolean: { input: boolean; output: boolean; }
  Int: { input: number; output: number; }
  Float: { input: number; output: number; }
  Date: { input: any; output: any; }
  DateTime: { input: any; output: any; }
};

export type Account = {
  __typename?: 'Account';
  accountType: AccountType;
  balance: Scalars['Float']['output'];
  id: Scalars['ID']['output'];
  institution: Scalars['String']['output'];
  isExternal: Scalars['Boolean']['output'];
};

export type AccountContribution = {
  __typename?: 'AccountContribution';
  accountType: AccountType;
  annualLimit: Scalars['Float']['output'];
  contributed: Scalars['Float']['output'];
  daysUntilDeadline?: Maybe<Scalars['Int']['output']>;
  deadline?: Maybe<Scalars['Date']['output']>;
  isOverContributed: Scalars['Boolean']['output'];
  lifetimeCap?: Maybe<Scalars['Float']['output']>;
  overAmount?: Maybe<Scalars['Float']['output']>;
  penaltyPerMonth?: Maybe<Scalars['Float']['output']>;
  remaining: Scalars['Float']['output'];
};

export type AccountType =
  | 'FHSA'
  | 'NON_REG'
  | 'RESP'
  | 'RRSP'
  | 'TFSA';

export type ActionItem = {
  __typename?: 'ActionItem';
  alert?: Maybe<Alert>;
  client: Client;
  createdAt: Scalars['DateTime']['output'];
  dueDate?: Maybe<Scalars['Date']['output']>;
  id: Scalars['ID']['output'];
  resolutionNote?: Maybe<Scalars['String']['output']>;
  resolvedAt?: Maybe<Scalars['DateTime']['output']>;
  status: ActionItemStatus;
  text: Scalars['String']['output'];
};

export type ActionItemStatus =
  | 'CLOSED'
  | 'DONE'
  | 'IN_PROGRESS'
  | 'PENDING';

export type Advisor = {
  __typename?: 'Advisor';
  email: Scalars['String']['output'];
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
  role: Scalars['String']['output'];
};

export type AdvisorNote = {
  __typename?: 'AdvisorNote';
  date: Scalars['Date']['output'];
  id: Scalars['ID']['output'];
  text: Scalars['String']['output'];
};

export type Alert = {
  __typename?: 'Alert';
  category: Scalars['String']['output'];
  client: Client;
  conditionKey: Scalars['String']['output'];
  createdAt: Scalars['DateTime']['output'];
  draftMessage?: Maybe<Scalars['String']['output']>;
  id: Scalars['ID']['output'];
  linkedActionItems: Array<ActionItem>;
  severity: AlertSeverity;
  snoozedUntil?: Maybe<Scalars['DateTime']['output']>;
  status: AlertStatus;
  summary: Scalars['String']['output'];
  updatedAt: Scalars['DateTime']['output'];
};

export type AlertEvent = {
  __typename?: 'AlertEvent';
  alert: Alert;
  type: AlertEventType;
};

export type AlertEventType =
  | 'CLOSED'
  | 'CREATED'
  | 'UPDATED';

export type AlertFilter = {
  clientId?: InputMaybe<Scalars['ID']['input']>;
  severity?: InputMaybe<AlertSeverity>;
  status?: InputMaybe<AlertStatus>;
};

export type AlertSeverity =
  | 'ADVISORY'
  | 'CRITICAL'
  | 'INFO'
  | 'URGENT';

export type AlertStatus =
  | 'ACTED'
  | 'CLOSED'
  | 'OPEN'
  | 'SNOOZED';

export type Client = {
  __typename?: 'Client';
  accounts: Array<Account>;
  actionItems: Array<ActionItem>;
  alerts: Array<Alert>;
  aum: Scalars['Float']['output'];
  dateOfBirth: Scalars['Date']['output'];
  email: Scalars['String']['output'];
  externalAccounts: Array<Account>;
  goals: Array<Goal>;
  health: HealthStatus;
  household?: Maybe<Household>;
  id: Scalars['ID']['output'];
  lastMeeting: Scalars['Date']['output'];
  name: Scalars['String']['output'];
  notes: Array<AdvisorNote>;
};

export type ContributionSummary = {
  __typename?: 'ContributionSummary';
  accounts: Array<AccountContribution>;
  clientId: Scalars['ID']['output'];
  taxYear: Scalars['Int']['output'];
};

export type CreateActionItemInput = {
  alertId?: InputMaybe<Scalars['ID']['input']>;
  clientId: Scalars['ID']['input'];
  dueDate?: InputMaybe<Scalars['Date']['input']>;
  text: Scalars['String']['input'];
};

export type Goal = {
  __typename?: 'Goal';
  id: Scalars['ID']['output'];
  name: Scalars['String']['output'];
  progressPct: Scalars['Int']['output'];
  status: GoalStatus;
  targetAmount?: Maybe<Scalars['Float']['output']>;
  targetDate?: Maybe<Scalars['Date']['output']>;
};

export type GoalStatus =
  | 'AHEAD'
  | 'BEHIND'
  | 'ON_TRACK';

export type HealthStatus =
  | 'GREEN'
  | 'RED'
  | 'YELLOW';

export type Household = {
  __typename?: 'Household';
  id: Scalars['ID']['output'];
  members: Array<Client>;
  name: Scalars['String']['output'];
};

export type Mutation = {
  __typename?: 'Mutation';
  acknowledgeAlert: Alert;
  addNote: AdvisorNote;
  createActionItem: ActionItem;
  runMorningSweep: SweepResult;
  sendAlert: Alert;
  snoozeAlert: Alert;
  trackAlert: Alert;
  updateActionItem: ActionItem;
};


export type MutationAcknowledgeAlertArgs = {
  alertId: Scalars['ID']['input'];
};


export type MutationAddNoteArgs = {
  clientId: Scalars['ID']['input'];
  text: Scalars['String']['input'];
};


export type MutationCreateActionItemArgs = {
  input: CreateActionItemInput;
};


export type MutationRunMorningSweepArgs = {
  advisorId: Scalars['ID']['input'];
};


export type MutationSendAlertArgs = {
  alertId: Scalars['ID']['input'];
  message?: InputMaybe<Scalars['String']['input']>;
};


export type MutationSnoozeAlertArgs = {
  alertId: Scalars['ID']['input'];
  until?: InputMaybe<Scalars['DateTime']['input']>;
};


export type MutationTrackAlertArgs = {
  actionItemText: Scalars['String']['input'];
  alertId: Scalars['ID']['input'];
};


export type MutationUpdateActionItemArgs = {
  id: Scalars['ID']['input'];
  input: UpdateActionItemInput;
};

export type Query = {
  __typename?: 'Query';
  actionItems: Array<ActionItem>;
  advisor: Advisor;
  alert: Alert;
  alerts: Array<Alert>;
  client: Client;
  clients: Array<Client>;
  contributionSummary: ContributionSummary;
  transfers: Array<Transfer>;
};


export type QueryActionItemsArgs = {
  clientId?: InputMaybe<Scalars['ID']['input']>;
};


export type QueryAdvisorArgs = {
  id: Scalars['ID']['input'];
};


export type QueryAlertArgs = {
  id: Scalars['ID']['input'];
};


export type QueryAlertsArgs = {
  advisorId: Scalars['ID']['input'];
  filter?: InputMaybe<AlertFilter>;
};


export type QueryClientArgs = {
  id: Scalars['ID']['input'];
};


export type QueryClientsArgs = {
  advisorId: Scalars['ID']['input'];
};


export type QueryContributionSummaryArgs = {
  clientId: Scalars['ID']['input'];
  taxYear: Scalars['Int']['input'];
};


export type QueryTransfersArgs = {
  advisorId: Scalars['ID']['input'];
};

export type Subscription = {
  __typename?: 'Subscription';
  alertFeed: AlertEvent;
};


export type SubscriptionAlertFeedArgs = {
  advisorId: Scalars['ID']['input'];
};

export type SweepResult = {
  __typename?: 'SweepResult';
  alertsGenerated: Scalars['Int']['output'];
  alertsSkipped: Scalars['Int']['output'];
  alertsUpdated: Scalars['Int']['output'];
  duration: Scalars['String']['output'];
};

export type Transfer = {
  __typename?: 'Transfer';
  accountType: AccountType;
  amount: Scalars['Float']['output'];
  client: Client;
  daysInCurrentStage: Scalars['Int']['output'];
  id: Scalars['ID']['output'];
  initiatedAt: Scalars['Date']['output'];
  isStuck: Scalars['Boolean']['output'];
  sourceInstitution: Scalars['String']['output'];
  status: TransferStatus;
};

export type TransferStatus =
  | 'DOCUMENTS_SUBMITTED'
  | 'INITIATED'
  | 'INVESTED'
  | 'IN_REVIEW'
  | 'IN_TRANSIT'
  | 'RECEIVED';

export type UpdateActionItemInput = {
  dueDate?: InputMaybe<Scalars['Date']['input']>;
  status?: InputMaybe<ActionItemStatus>;
  text?: InputMaybe<Scalars['String']['input']>;
};

export type GetAlertsQueryVariables = Exact<{
  advisorId: Scalars['ID']['input'];
  filter?: InputMaybe<AlertFilter>;
}>;


export type GetAlertsQuery = { __typename?: 'Query', alerts: Array<{ __typename?: 'Alert', id: string, conditionKey: string, severity: AlertSeverity, category: string, status: AlertStatus, snoozedUntil?: any | null, summary: string, draftMessage?: string | null, createdAt: any, updatedAt: any, client: { __typename?: 'Client', id: string, name: string }, linkedActionItems: Array<{ __typename?: 'ActionItem', id: string, text: string, status: ActionItemStatus }> }> };

export type SendAlertMutationVariables = Exact<{
  alertId: Scalars['ID']['input'];
  message?: InputMaybe<Scalars['String']['input']>;
}>;


export type SendAlertMutation = { __typename?: 'Mutation', sendAlert: { __typename?: 'Alert', id: string, status: AlertStatus, summary: string, draftMessage?: string | null, updatedAt: any, linkedActionItems: Array<{ __typename?: 'ActionItem', id: string, text: string, status: ActionItemStatus }> } };

export type TrackAlertMutationVariables = Exact<{
  alertId: Scalars['ID']['input'];
  actionItemText: Scalars['String']['input'];
}>;


export type TrackAlertMutation = { __typename?: 'Mutation', trackAlert: { __typename?: 'Alert', id: string, status: AlertStatus, updatedAt: any, linkedActionItems: Array<{ __typename?: 'ActionItem', id: string, text: string, status: ActionItemStatus }> } };

export type SnoozeAlertMutationVariables = Exact<{
  alertId: Scalars['ID']['input'];
  until?: InputMaybe<Scalars['DateTime']['input']>;
}>;


export type SnoozeAlertMutation = { __typename?: 'Mutation', snoozeAlert: { __typename?: 'Alert', id: string, status: AlertStatus, snoozedUntil?: any | null, updatedAt: any } };

export type AcknowledgeAlertMutationVariables = Exact<{
  alertId: Scalars['ID']['input'];
}>;


export type AcknowledgeAlertMutation = { __typename?: 'Mutation', acknowledgeAlert: { __typename?: 'Alert', id: string, status: AlertStatus, updatedAt: any } };

export type AlertFeedSubscriptionVariables = Exact<{
  advisorId: Scalars['ID']['input'];
}>;


export type AlertFeedSubscription = { __typename?: 'Subscription', alertFeed: { __typename?: 'AlertEvent', type: AlertEventType, alert: { __typename?: 'Alert', id: string, conditionKey: string, severity: AlertSeverity, category: string, status: AlertStatus, snoozedUntil?: any | null, summary: string, draftMessage?: string | null, createdAt: any, updatedAt: any, client: { __typename?: 'Client', id: string, name: string }, linkedActionItems: Array<{ __typename?: 'ActionItem', id: string, text: string, status: ActionItemStatus }> } } };

export type GetAdvisorQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetAdvisorQuery = { __typename?: 'Query', advisor: { __typename?: 'Advisor', id: string, name: string, email: string, role: string } };

export type GetClientsQueryVariables = Exact<{
  advisorId: Scalars['ID']['input'];
}>;


export type GetClientsQuery = { __typename?: 'Query', clients: Array<{ __typename?: 'Client', id: string, name: string, email: string, dateOfBirth: any, aum: number, lastMeeting: any, health: HealthStatus, accounts: Array<{ __typename?: 'Account', id: string, accountType: AccountType }>, alerts: Array<{ __typename?: 'Alert', id: string }> }> };

export type GetClientQueryVariables = Exact<{
  id: Scalars['ID']['input'];
}>;


export type GetClientQuery = { __typename?: 'Query', client: { __typename?: 'Client', id: string, name: string, email: string, dateOfBirth: any, aum: number, lastMeeting: any, health: HealthStatus, household?: { __typename?: 'Household', id: string, name: string, members: Array<{ __typename?: 'Client', id: string, name: string }> } | null, accounts: Array<{ __typename?: 'Account', id: string, accountType: AccountType, institution: string, balance: number, isExternal: boolean }>, externalAccounts: Array<{ __typename?: 'Account', id: string, accountType: AccountType, institution: string, balance: number, isExternal: boolean }>, alerts: Array<{ __typename?: 'Alert', id: string, severity: AlertSeverity, category: string, summary: string, status: AlertStatus }>, actionItems: Array<{ __typename?: 'ActionItem', id: string, text: string, status: ActionItemStatus, dueDate?: any | null, createdAt: any, resolvedAt?: any | null, resolutionNote?: string | null }>, goals: Array<{ __typename?: 'Goal', id: string, name: string, targetAmount?: number | null, targetDate?: any | null, progressPct: number, status: GoalStatus }>, notes: Array<{ __typename?: 'AdvisorNote', id: string, date: any, text: string }> } };

export type GetContributionSummaryQueryVariables = Exact<{
  clientId: Scalars['ID']['input'];
  taxYear: Scalars['Int']['input'];
}>;


export type GetContributionSummaryQuery = { __typename?: 'Query', contributionSummary: { __typename?: 'ContributionSummary', clientId: string, taxYear: number, accounts: Array<{ __typename?: 'AccountContribution', accountType: AccountType, annualLimit: number, lifetimeCap?: number | null, contributed: number, remaining: number, isOverContributed: boolean, overAmount?: number | null, penaltyPerMonth?: number | null, deadline?: any | null, daysUntilDeadline?: number | null }> } };

export type GetActionItemsQueryVariables = Exact<{
  clientId?: InputMaybe<Scalars['ID']['input']>;
}>;


export type GetActionItemsQuery = { __typename?: 'Query', actionItems: Array<{ __typename?: 'ActionItem', id: string, text: string, status: ActionItemStatus, dueDate?: any | null, createdAt: any, resolvedAt?: any | null, resolutionNote?: string | null, client: { __typename?: 'Client', id: string, name: string }, alert?: { __typename?: 'Alert', id: string } | null }> };

export type AddNoteMutationVariables = Exact<{
  clientId: Scalars['ID']['input'];
  text: Scalars['String']['input'];
}>;


export type AddNoteMutation = { __typename?: 'Mutation', addNote: { __typename?: 'AdvisorNote', id: string, date: any, text: string } };

export type RunMorningSweepMutationVariables = Exact<{
  advisorId: Scalars['ID']['input'];
}>;


export type RunMorningSweepMutation = { __typename?: 'Mutation', runMorningSweep: { __typename?: 'SweepResult', alertsGenerated: number, alertsUpdated: number, alertsSkipped: number, duration: string } };

export type GetTransfersQueryVariables = Exact<{
  advisorId: Scalars['ID']['input'];
}>;


export type GetTransfersQuery = { __typename?: 'Query', transfers: Array<{ __typename?: 'Transfer', id: string, sourceInstitution: string, accountType: AccountType, amount: number, status: TransferStatus, initiatedAt: any, daysInCurrentStage: number, isStuck: boolean, client: { __typename?: 'Client', id: string, name: string } }> };


export const GetAlertsDocument = gql`
    query GetAlerts($advisorId: ID!, $filter: AlertFilter) {
  alerts(advisorId: $advisorId, filter: $filter) {
    id
    conditionKey
    client {
      id
      name
    }
    severity
    category
    status
    snoozedUntil
    summary
    draftMessage
    linkedActionItems {
      id
      text
      status
    }
    createdAt
    updatedAt
  }
}
    `;

/**
 * __useGetAlertsQuery__
 *
 * To run a query within a React component, call `useGetAlertsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetAlertsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetAlertsQuery({
 *   variables: {
 *      advisorId: // value for 'advisorId'
 *      filter: // value for 'filter'
 *   },
 * });
 */
export function useGetAlertsQuery(baseOptions: Apollo.QueryHookOptions<GetAlertsQuery, GetAlertsQueryVariables> & ({ variables: GetAlertsQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetAlertsQuery, GetAlertsQueryVariables>(GetAlertsDocument, options);
      }
export function useGetAlertsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetAlertsQuery, GetAlertsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetAlertsQuery, GetAlertsQueryVariables>(GetAlertsDocument, options);
        }
// @ts-ignore
export function useGetAlertsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetAlertsQuery, GetAlertsQueryVariables>): Apollo.UseSuspenseQueryResult<GetAlertsQuery, GetAlertsQueryVariables>;
export function useGetAlertsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAlertsQuery, GetAlertsQueryVariables>): Apollo.UseSuspenseQueryResult<GetAlertsQuery | undefined, GetAlertsQueryVariables>;
export function useGetAlertsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAlertsQuery, GetAlertsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAlertsQuery, GetAlertsQueryVariables>(GetAlertsDocument, options);
        }
export type GetAlertsQueryHookResult = ReturnType<typeof useGetAlertsQuery>;
export type GetAlertsLazyQueryHookResult = ReturnType<typeof useGetAlertsLazyQuery>;
export type GetAlertsSuspenseQueryHookResult = ReturnType<typeof useGetAlertsSuspenseQuery>;
export type GetAlertsQueryResult = Apollo.QueryResult<GetAlertsQuery, GetAlertsQueryVariables>;
export const SendAlertDocument = gql`
    mutation SendAlert($alertId: ID!, $message: String) {
  sendAlert(alertId: $alertId, message: $message) {
    id
    status
    summary
    draftMessage
    linkedActionItems {
      id
      text
      status
    }
    updatedAt
  }
}
    `;
export type SendAlertMutationFn = Apollo.MutationFunction<SendAlertMutation, SendAlertMutationVariables>;

/**
 * __useSendAlertMutation__
 *
 * To run a mutation, you first call `useSendAlertMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useSendAlertMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [sendAlertMutation, { data, loading, error }] = useSendAlertMutation({
 *   variables: {
 *      alertId: // value for 'alertId'
 *      message: // value for 'message'
 *   },
 * });
 */
export function useSendAlertMutation(baseOptions?: Apollo.MutationHookOptions<SendAlertMutation, SendAlertMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<SendAlertMutation, SendAlertMutationVariables>(SendAlertDocument, options);
      }
export type SendAlertMutationHookResult = ReturnType<typeof useSendAlertMutation>;
export type SendAlertMutationResult = Apollo.MutationResult<SendAlertMutation>;
export type SendAlertMutationOptions = Apollo.BaseMutationOptions<SendAlertMutation, SendAlertMutationVariables>;
export const TrackAlertDocument = gql`
    mutation TrackAlert($alertId: ID!, $actionItemText: String!) {
  trackAlert(alertId: $alertId, actionItemText: $actionItemText) {
    id
    status
    linkedActionItems {
      id
      text
      status
    }
    updatedAt
  }
}
    `;
export type TrackAlertMutationFn = Apollo.MutationFunction<TrackAlertMutation, TrackAlertMutationVariables>;

/**
 * __useTrackAlertMutation__
 *
 * To run a mutation, you first call `useTrackAlertMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useTrackAlertMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [trackAlertMutation, { data, loading, error }] = useTrackAlertMutation({
 *   variables: {
 *      alertId: // value for 'alertId'
 *      actionItemText: // value for 'actionItemText'
 *   },
 * });
 */
export function useTrackAlertMutation(baseOptions?: Apollo.MutationHookOptions<TrackAlertMutation, TrackAlertMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<TrackAlertMutation, TrackAlertMutationVariables>(TrackAlertDocument, options);
      }
export type TrackAlertMutationHookResult = ReturnType<typeof useTrackAlertMutation>;
export type TrackAlertMutationResult = Apollo.MutationResult<TrackAlertMutation>;
export type TrackAlertMutationOptions = Apollo.BaseMutationOptions<TrackAlertMutation, TrackAlertMutationVariables>;
export const SnoozeAlertDocument = gql`
    mutation SnoozeAlert($alertId: ID!, $until: DateTime) {
  snoozeAlert(alertId: $alertId, until: $until) {
    id
    status
    snoozedUntil
    updatedAt
  }
}
    `;
export type SnoozeAlertMutationFn = Apollo.MutationFunction<SnoozeAlertMutation, SnoozeAlertMutationVariables>;

/**
 * __useSnoozeAlertMutation__
 *
 * To run a mutation, you first call `useSnoozeAlertMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useSnoozeAlertMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [snoozeAlertMutation, { data, loading, error }] = useSnoozeAlertMutation({
 *   variables: {
 *      alertId: // value for 'alertId'
 *      until: // value for 'until'
 *   },
 * });
 */
export function useSnoozeAlertMutation(baseOptions?: Apollo.MutationHookOptions<SnoozeAlertMutation, SnoozeAlertMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<SnoozeAlertMutation, SnoozeAlertMutationVariables>(SnoozeAlertDocument, options);
      }
export type SnoozeAlertMutationHookResult = ReturnType<typeof useSnoozeAlertMutation>;
export type SnoozeAlertMutationResult = Apollo.MutationResult<SnoozeAlertMutation>;
export type SnoozeAlertMutationOptions = Apollo.BaseMutationOptions<SnoozeAlertMutation, SnoozeAlertMutationVariables>;
export const AcknowledgeAlertDocument = gql`
    mutation AcknowledgeAlert($alertId: ID!) {
  acknowledgeAlert(alertId: $alertId) {
    id
    status
    updatedAt
  }
}
    `;
export type AcknowledgeAlertMutationFn = Apollo.MutationFunction<AcknowledgeAlertMutation, AcknowledgeAlertMutationVariables>;

/**
 * __useAcknowledgeAlertMutation__
 *
 * To run a mutation, you first call `useAcknowledgeAlertMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useAcknowledgeAlertMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [acknowledgeAlertMutation, { data, loading, error }] = useAcknowledgeAlertMutation({
 *   variables: {
 *      alertId: // value for 'alertId'
 *   },
 * });
 */
export function useAcknowledgeAlertMutation(baseOptions?: Apollo.MutationHookOptions<AcknowledgeAlertMutation, AcknowledgeAlertMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<AcknowledgeAlertMutation, AcknowledgeAlertMutationVariables>(AcknowledgeAlertDocument, options);
      }
export type AcknowledgeAlertMutationHookResult = ReturnType<typeof useAcknowledgeAlertMutation>;
export type AcknowledgeAlertMutationResult = Apollo.MutationResult<AcknowledgeAlertMutation>;
export type AcknowledgeAlertMutationOptions = Apollo.BaseMutationOptions<AcknowledgeAlertMutation, AcknowledgeAlertMutationVariables>;
export const AlertFeedDocument = gql`
    subscription AlertFeed($advisorId: ID!) {
  alertFeed(advisorId: $advisorId) {
    type
    alert {
      id
      conditionKey
      client {
        id
        name
      }
      severity
      category
      status
      snoozedUntil
      summary
      draftMessage
      linkedActionItems {
        id
        text
        status
      }
      createdAt
      updatedAt
    }
  }
}
    `;

/**
 * __useAlertFeedSubscription__
 *
 * To run a query within a React component, call `useAlertFeedSubscription` and pass it any options that fit your needs.
 * When your component renders, `useAlertFeedSubscription` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the subscription, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useAlertFeedSubscription({
 *   variables: {
 *      advisorId: // value for 'advisorId'
 *   },
 * });
 */
export function useAlertFeedSubscription(baseOptions: Apollo.SubscriptionHookOptions<AlertFeedSubscription, AlertFeedSubscriptionVariables> & ({ variables: AlertFeedSubscriptionVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useSubscription<AlertFeedSubscription, AlertFeedSubscriptionVariables>(AlertFeedDocument, options);
      }
export type AlertFeedSubscriptionHookResult = ReturnType<typeof useAlertFeedSubscription>;
export type AlertFeedSubscriptionResult = Apollo.SubscriptionResult<AlertFeedSubscription>;
export const GetAdvisorDocument = gql`
    query GetAdvisor($id: ID!) {
  advisor(id: $id) {
    id
    name
    email
    role
  }
}
    `;

/**
 * __useGetAdvisorQuery__
 *
 * To run a query within a React component, call `useGetAdvisorQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetAdvisorQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetAdvisorQuery({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useGetAdvisorQuery(baseOptions: Apollo.QueryHookOptions<GetAdvisorQuery, GetAdvisorQueryVariables> & ({ variables: GetAdvisorQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetAdvisorQuery, GetAdvisorQueryVariables>(GetAdvisorDocument, options);
      }
export function useGetAdvisorLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetAdvisorQuery, GetAdvisorQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetAdvisorQuery, GetAdvisorQueryVariables>(GetAdvisorDocument, options);
        }
// @ts-ignore
export function useGetAdvisorSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetAdvisorQuery, GetAdvisorQueryVariables>): Apollo.UseSuspenseQueryResult<GetAdvisorQuery, GetAdvisorQueryVariables>;
export function useGetAdvisorSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAdvisorQuery, GetAdvisorQueryVariables>): Apollo.UseSuspenseQueryResult<GetAdvisorQuery | undefined, GetAdvisorQueryVariables>;
export function useGetAdvisorSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetAdvisorQuery, GetAdvisorQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetAdvisorQuery, GetAdvisorQueryVariables>(GetAdvisorDocument, options);
        }
export type GetAdvisorQueryHookResult = ReturnType<typeof useGetAdvisorQuery>;
export type GetAdvisorLazyQueryHookResult = ReturnType<typeof useGetAdvisorLazyQuery>;
export type GetAdvisorSuspenseQueryHookResult = ReturnType<typeof useGetAdvisorSuspenseQuery>;
export type GetAdvisorQueryResult = Apollo.QueryResult<GetAdvisorQuery, GetAdvisorQueryVariables>;
export const GetClientsDocument = gql`
    query GetClients($advisorId: ID!) {
  clients(advisorId: $advisorId) {
    id
    name
    email
    dateOfBirth
    aum
    lastMeeting
    health
    accounts {
      id
      accountType
    }
    alerts {
      id
    }
  }
}
    `;

/**
 * __useGetClientsQuery__
 *
 * To run a query within a React component, call `useGetClientsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetClientsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetClientsQuery({
 *   variables: {
 *      advisorId: // value for 'advisorId'
 *   },
 * });
 */
export function useGetClientsQuery(baseOptions: Apollo.QueryHookOptions<GetClientsQuery, GetClientsQueryVariables> & ({ variables: GetClientsQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetClientsQuery, GetClientsQueryVariables>(GetClientsDocument, options);
      }
export function useGetClientsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetClientsQuery, GetClientsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetClientsQuery, GetClientsQueryVariables>(GetClientsDocument, options);
        }
// @ts-ignore
export function useGetClientsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetClientsQuery, GetClientsQueryVariables>): Apollo.UseSuspenseQueryResult<GetClientsQuery, GetClientsQueryVariables>;
export function useGetClientsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetClientsQuery, GetClientsQueryVariables>): Apollo.UseSuspenseQueryResult<GetClientsQuery | undefined, GetClientsQueryVariables>;
export function useGetClientsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetClientsQuery, GetClientsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetClientsQuery, GetClientsQueryVariables>(GetClientsDocument, options);
        }
export type GetClientsQueryHookResult = ReturnType<typeof useGetClientsQuery>;
export type GetClientsLazyQueryHookResult = ReturnType<typeof useGetClientsLazyQuery>;
export type GetClientsSuspenseQueryHookResult = ReturnType<typeof useGetClientsSuspenseQuery>;
export type GetClientsQueryResult = Apollo.QueryResult<GetClientsQuery, GetClientsQueryVariables>;
export const GetClientDocument = gql`
    query GetClient($id: ID!) {
  client(id: $id) {
    id
    name
    email
    dateOfBirth
    aum
    lastMeeting
    health
    household {
      id
      name
      members {
        id
        name
      }
    }
    accounts {
      id
      accountType
      institution
      balance
      isExternal
    }
    externalAccounts {
      id
      accountType
      institution
      balance
      isExternal
    }
    alerts {
      id
      severity
      category
      summary
      status
    }
    actionItems {
      id
      text
      status
      dueDate
      createdAt
      resolvedAt
      resolutionNote
    }
    goals {
      id
      name
      targetAmount
      targetDate
      progressPct
      status
    }
    notes {
      id
      date
      text
    }
  }
}
    `;

/**
 * __useGetClientQuery__
 *
 * To run a query within a React component, call `useGetClientQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetClientQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetClientQuery({
 *   variables: {
 *      id: // value for 'id'
 *   },
 * });
 */
export function useGetClientQuery(baseOptions: Apollo.QueryHookOptions<GetClientQuery, GetClientQueryVariables> & ({ variables: GetClientQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetClientQuery, GetClientQueryVariables>(GetClientDocument, options);
      }
export function useGetClientLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetClientQuery, GetClientQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetClientQuery, GetClientQueryVariables>(GetClientDocument, options);
        }
// @ts-ignore
export function useGetClientSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetClientQuery, GetClientQueryVariables>): Apollo.UseSuspenseQueryResult<GetClientQuery, GetClientQueryVariables>;
export function useGetClientSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetClientQuery, GetClientQueryVariables>): Apollo.UseSuspenseQueryResult<GetClientQuery | undefined, GetClientQueryVariables>;
export function useGetClientSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetClientQuery, GetClientQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetClientQuery, GetClientQueryVariables>(GetClientDocument, options);
        }
export type GetClientQueryHookResult = ReturnType<typeof useGetClientQuery>;
export type GetClientLazyQueryHookResult = ReturnType<typeof useGetClientLazyQuery>;
export type GetClientSuspenseQueryHookResult = ReturnType<typeof useGetClientSuspenseQuery>;
export type GetClientQueryResult = Apollo.QueryResult<GetClientQuery, GetClientQueryVariables>;
export const GetContributionSummaryDocument = gql`
    query GetContributionSummary($clientId: ID!, $taxYear: Int!) {
  contributionSummary(clientId: $clientId, taxYear: $taxYear) {
    clientId
    taxYear
    accounts {
      accountType
      annualLimit
      lifetimeCap
      contributed
      remaining
      isOverContributed
      overAmount
      penaltyPerMonth
      deadline
      daysUntilDeadline
    }
  }
}
    `;

/**
 * __useGetContributionSummaryQuery__
 *
 * To run a query within a React component, call `useGetContributionSummaryQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetContributionSummaryQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetContributionSummaryQuery({
 *   variables: {
 *      clientId: // value for 'clientId'
 *      taxYear: // value for 'taxYear'
 *   },
 * });
 */
export function useGetContributionSummaryQuery(baseOptions: Apollo.QueryHookOptions<GetContributionSummaryQuery, GetContributionSummaryQueryVariables> & ({ variables: GetContributionSummaryQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetContributionSummaryQuery, GetContributionSummaryQueryVariables>(GetContributionSummaryDocument, options);
      }
export function useGetContributionSummaryLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetContributionSummaryQuery, GetContributionSummaryQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetContributionSummaryQuery, GetContributionSummaryQueryVariables>(GetContributionSummaryDocument, options);
        }
// @ts-ignore
export function useGetContributionSummarySuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetContributionSummaryQuery, GetContributionSummaryQueryVariables>): Apollo.UseSuspenseQueryResult<GetContributionSummaryQuery, GetContributionSummaryQueryVariables>;
export function useGetContributionSummarySuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetContributionSummaryQuery, GetContributionSummaryQueryVariables>): Apollo.UseSuspenseQueryResult<GetContributionSummaryQuery | undefined, GetContributionSummaryQueryVariables>;
export function useGetContributionSummarySuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetContributionSummaryQuery, GetContributionSummaryQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetContributionSummaryQuery, GetContributionSummaryQueryVariables>(GetContributionSummaryDocument, options);
        }
export type GetContributionSummaryQueryHookResult = ReturnType<typeof useGetContributionSummaryQuery>;
export type GetContributionSummaryLazyQueryHookResult = ReturnType<typeof useGetContributionSummaryLazyQuery>;
export type GetContributionSummarySuspenseQueryHookResult = ReturnType<typeof useGetContributionSummarySuspenseQuery>;
export type GetContributionSummaryQueryResult = Apollo.QueryResult<GetContributionSummaryQuery, GetContributionSummaryQueryVariables>;
export const GetActionItemsDocument = gql`
    query GetActionItems($clientId: ID) {
  actionItems(clientId: $clientId) {
    id
    client {
      id
      name
    }
    alert {
      id
    }
    text
    status
    dueDate
    createdAt
    resolvedAt
    resolutionNote
  }
}
    `;

/**
 * __useGetActionItemsQuery__
 *
 * To run a query within a React component, call `useGetActionItemsQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetActionItemsQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetActionItemsQuery({
 *   variables: {
 *      clientId: // value for 'clientId'
 *   },
 * });
 */
export function useGetActionItemsQuery(baseOptions?: Apollo.QueryHookOptions<GetActionItemsQuery, GetActionItemsQueryVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetActionItemsQuery, GetActionItemsQueryVariables>(GetActionItemsDocument, options);
      }
export function useGetActionItemsLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetActionItemsQuery, GetActionItemsQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetActionItemsQuery, GetActionItemsQueryVariables>(GetActionItemsDocument, options);
        }
// @ts-ignore
export function useGetActionItemsSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetActionItemsQuery, GetActionItemsQueryVariables>): Apollo.UseSuspenseQueryResult<GetActionItemsQuery, GetActionItemsQueryVariables>;
export function useGetActionItemsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetActionItemsQuery, GetActionItemsQueryVariables>): Apollo.UseSuspenseQueryResult<GetActionItemsQuery | undefined, GetActionItemsQueryVariables>;
export function useGetActionItemsSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetActionItemsQuery, GetActionItemsQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetActionItemsQuery, GetActionItemsQueryVariables>(GetActionItemsDocument, options);
        }
export type GetActionItemsQueryHookResult = ReturnType<typeof useGetActionItemsQuery>;
export type GetActionItemsLazyQueryHookResult = ReturnType<typeof useGetActionItemsLazyQuery>;
export type GetActionItemsSuspenseQueryHookResult = ReturnType<typeof useGetActionItemsSuspenseQuery>;
export type GetActionItemsQueryResult = Apollo.QueryResult<GetActionItemsQuery, GetActionItemsQueryVariables>;
export const AddNoteDocument = gql`
    mutation AddNote($clientId: ID!, $text: String!) {
  addNote(clientId: $clientId, text: $text) {
    id
    date
    text
  }
}
    `;
export type AddNoteMutationFn = Apollo.MutationFunction<AddNoteMutation, AddNoteMutationVariables>;

/**
 * __useAddNoteMutation__
 *
 * To run a mutation, you first call `useAddNoteMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useAddNoteMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [addNoteMutation, { data, loading, error }] = useAddNoteMutation({
 *   variables: {
 *      clientId: // value for 'clientId'
 *      text: // value for 'text'
 *   },
 * });
 */
export function useAddNoteMutation(baseOptions?: Apollo.MutationHookOptions<AddNoteMutation, AddNoteMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<AddNoteMutation, AddNoteMutationVariables>(AddNoteDocument, options);
      }
export type AddNoteMutationHookResult = ReturnType<typeof useAddNoteMutation>;
export type AddNoteMutationResult = Apollo.MutationResult<AddNoteMutation>;
export type AddNoteMutationOptions = Apollo.BaseMutationOptions<AddNoteMutation, AddNoteMutationVariables>;
export const RunMorningSweepDocument = gql`
    mutation RunMorningSweep($advisorId: ID!) {
  runMorningSweep(advisorId: $advisorId) {
    alertsGenerated
    alertsUpdated
    alertsSkipped
    duration
  }
}
    `;
export type RunMorningSweepMutationFn = Apollo.MutationFunction<RunMorningSweepMutation, RunMorningSweepMutationVariables>;

/**
 * __useRunMorningSweepMutation__
 *
 * To run a mutation, you first call `useRunMorningSweepMutation` within a React component and pass it any options that fit your needs.
 * When your component renders, `useRunMorningSweepMutation` returns a tuple that includes:
 * - A mutate function that you can call at any time to execute the mutation
 * - An object with fields that represent the current status of the mutation's execution
 *
 * @param baseOptions options that will be passed into the mutation, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options-2;
 *
 * @example
 * const [runMorningSweepMutation, { data, loading, error }] = useRunMorningSweepMutation({
 *   variables: {
 *      advisorId: // value for 'advisorId'
 *   },
 * });
 */
export function useRunMorningSweepMutation(baseOptions?: Apollo.MutationHookOptions<RunMorningSweepMutation, RunMorningSweepMutationVariables>) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useMutation<RunMorningSweepMutation, RunMorningSweepMutationVariables>(RunMorningSweepDocument, options);
      }
export type RunMorningSweepMutationHookResult = ReturnType<typeof useRunMorningSweepMutation>;
export type RunMorningSweepMutationResult = Apollo.MutationResult<RunMorningSweepMutation>;
export type RunMorningSweepMutationOptions = Apollo.BaseMutationOptions<RunMorningSweepMutation, RunMorningSweepMutationVariables>;
export const GetTransfersDocument = gql`
    query GetTransfers($advisorId: ID!) {
  transfers(advisorId: $advisorId) {
    id
    client {
      id
      name
    }
    sourceInstitution
    accountType
    amount
    status
    initiatedAt
    daysInCurrentStage
    isStuck
  }
}
    `;

/**
 * __useGetTransfersQuery__
 *
 * To run a query within a React component, call `useGetTransfersQuery` and pass it any options that fit your needs.
 * When your component renders, `useGetTransfersQuery` returns an object from Apollo Client that contains loading, error, and data properties
 * you can use to render your UI.
 *
 * @param baseOptions options that will be passed into the query, supported options are listed on: https://www.apollographql.com/docs/react/api/react-hooks/#options;
 *
 * @example
 * const { data, loading, error } = useGetTransfersQuery({
 *   variables: {
 *      advisorId: // value for 'advisorId'
 *   },
 * });
 */
export function useGetTransfersQuery(baseOptions: Apollo.QueryHookOptions<GetTransfersQuery, GetTransfersQueryVariables> & ({ variables: GetTransfersQueryVariables; skip?: boolean; } | { skip: boolean; }) ) {
        const options = {...defaultOptions, ...baseOptions}
        return Apollo.useQuery<GetTransfersQuery, GetTransfersQueryVariables>(GetTransfersDocument, options);
      }
export function useGetTransfersLazyQuery(baseOptions?: Apollo.LazyQueryHookOptions<GetTransfersQuery, GetTransfersQueryVariables>) {
          const options = {...defaultOptions, ...baseOptions}
          return Apollo.useLazyQuery<GetTransfersQuery, GetTransfersQueryVariables>(GetTransfersDocument, options);
        }
// @ts-ignore
export function useGetTransfersSuspenseQuery(baseOptions?: Apollo.SuspenseQueryHookOptions<GetTransfersQuery, GetTransfersQueryVariables>): Apollo.UseSuspenseQueryResult<GetTransfersQuery, GetTransfersQueryVariables>;
export function useGetTransfersSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTransfersQuery, GetTransfersQueryVariables>): Apollo.UseSuspenseQueryResult<GetTransfersQuery | undefined, GetTransfersQueryVariables>;
export function useGetTransfersSuspenseQuery(baseOptions?: Apollo.SkipToken | Apollo.SuspenseQueryHookOptions<GetTransfersQuery, GetTransfersQueryVariables>) {
          const options = baseOptions === Apollo.skipToken ? baseOptions : {...defaultOptions, ...baseOptions}
          return Apollo.useSuspenseQuery<GetTransfersQuery, GetTransfersQueryVariables>(GetTransfersDocument, options);
        }
export type GetTransfersQueryHookResult = ReturnType<typeof useGetTransfersQuery>;
export type GetTransfersLazyQueryHookResult = ReturnType<typeof useGetTransfersLazyQuery>;
export type GetTransfersSuspenseQueryHookResult = ReturnType<typeof useGetTransfersSuspenseQuery>;
export type GetTransfersQueryResult = Apollo.QueryResult<GetTransfersQuery, GetTransfersQueryVariables>;