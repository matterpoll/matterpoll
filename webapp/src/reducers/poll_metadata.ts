import ActionTypes from '@/action_types';

import type {FetchPollMetadataAction} from '@/types/actions';
import type {PollMetadataMap} from '@/types/poll';

export const pollMetadata = (state: PollMetadataMap = {}, action: FetchPollMetadataAction): PollMetadataMap => {
    switch (action.type) {
    case ActionTypes.FETCH_POLL_METADATA:
        if (action.data) {
            const nextState = {...state};
            if (!action.data.poll_id) {
                return state;
            }
            nextState[action.data.poll_id] = action.data;
            return nextState;
        }
        return state;
    default:
        return state;
    }
};
