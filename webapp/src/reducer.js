import {combineReducers} from 'redux';

import {FETCH_VOTED_ANSWERS, REGIST_POST_TYPE_COMPONENT_ID} from './actions';

const postTypeComponent = (state = {}, action) => {
    switch (action.type) {
    case REGIST_POST_TYPE_COMPONENT_ID:
        if (action.data) {
            const nextState = {...state};
            if (!action.data.postTypeComponentId) {
                return state;
            }
            nextState['id'] = action.data.postTypeComponentId;
            return nextState;
        }
        return state;
    default:
        return state;
    }
}

const votedAnswers = (state = {}, action) => {
    switch (action.type) {
    case FETCH_VOTED_ANSWERS:
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

export default combineReducers({
    postTypeComponent,
    votedAnswers,
});