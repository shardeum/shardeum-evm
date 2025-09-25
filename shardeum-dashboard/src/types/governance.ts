export interface ProposalContent {
  typeUrl: string
  title: string
  description: string
  // For community spend proposals
  recipient?: string
  amount?: Array<{
    denom: string
    amount: string
  }>
  // For parameter change proposals
  changes?: Array<{
    subspace: string
    key: string
    value: string
  }>
}

export interface TallyResult {
  yes: string
  abstain: string
  no: string
  noWithVeto: string
}

export interface ProposalDeposit {
  proposalId: string
  depositor: string
  amount: Array<{
    denom: string
    amount: string
  }>
}

export interface ProposalVote {
  proposalId: string
  voter: string
  option: VoteOption
  options: Array<{
    option: VoteOption
    weight: string
  }>
}

export enum VoteOption {
  VOTE_OPTION_UNSPECIFIED = 0,
  VOTE_OPTION_YES = 1,
  VOTE_OPTION_ABSTAIN = 2,
  VOTE_OPTION_NO = 3,
  VOTE_OPTION_NO_WITH_VETO = 4
}

export enum ProposalStatus {
  PROPOSAL_STATUS_UNSPECIFIED = 0,
  PROPOSAL_STATUS_DEPOSIT_PERIOD = 1,
  PROPOSAL_STATUS_VOTING_PERIOD = 2,
  PROPOSAL_STATUS_PASSED = 3,
  PROPOSAL_STATUS_REJECTED = 4,
  PROPOSAL_STATUS_FAILED = 5
}

export interface GovernanceProposal {
  proposalId: string
  content: ProposalContent
  status: ProposalStatus
  finalTallyResult: TallyResult
  submitTime: string
  depositEndTime: string
  totalDeposit: Array<{
    denom: string
    amount: string
  }>
  votingStartTime: string
  votingEndTime: string
}

export interface GovernanceParams {
  minDeposit: Array<{
    denom: string
    amount: string
  }>
  maxDepositPeriod: string
  votingPeriod: string
  quorum: string
  threshold: string
  vetoThreshold: string
}

export type VoteOptionString = 'yes' | 'no' | 'abstain' | 'no_with_veto'

export function voteOptionStringToEnum(option: VoteOptionString): VoteOption {
  switch (option) {
    case 'yes':
      return VoteOption.VOTE_OPTION_YES
    case 'no':
      return VoteOption.VOTE_OPTION_NO
    case 'abstain':
      return VoteOption.VOTE_OPTION_ABSTAIN
    case 'no_with_veto':
      return VoteOption.VOTE_OPTION_NO_WITH_VETO
    default:
      return VoteOption.VOTE_OPTION_UNSPECIFIED
  }
}

export function getProposalStatusText(status: ProposalStatus): string {
  switch (status) {
    case ProposalStatus.PROPOSAL_STATUS_DEPOSIT_PERIOD:
      return 'Deposit Period'
    case ProposalStatus.PROPOSAL_STATUS_VOTING_PERIOD:
      return 'Voting'
    case ProposalStatus.PROPOSAL_STATUS_PASSED:
      return 'Passed'
    case ProposalStatus.PROPOSAL_STATUS_REJECTED:
      return 'Rejected'
    case ProposalStatus.PROPOSAL_STATUS_FAILED:
      return 'Failed'
    default:
      return 'Unknown'
  }
}

export function calculateVotePercentages(tally: TallyResult): {
  yes: number
  no: number
  abstain: number
  noWithVeto: number
  total: bigint
} {
  const yes = BigInt(tally.yes || '0')
  const no = BigInt(tally.no || '0')
  const abstain = BigInt(tally.abstain || '0')
  const noWithVeto = BigInt(tally.noWithVeto || '0')
  
  const total = yes + no + abstain + noWithVeto
  
  if (total === 0n) {
    return {
      yes: 0,
      no: 0,
      abstain: 0,
      noWithVeto: 0,
      total
    }
  }
  
  return {
    yes: Number((yes * 100n) / total),
    no: Number((no * 100n) / total),
    abstain: Number((abstain * 100n) / total),
    noWithVeto: Number((noWithVeto * 100n) / total),
    total
  }
}