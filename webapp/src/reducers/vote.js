import ActionTypes from 'action_types';

export const pollMetadata = (state = {}, action) => {
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
