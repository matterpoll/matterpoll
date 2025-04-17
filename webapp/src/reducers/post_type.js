import ActionTypes from '@/action_types';

export const postTypeComponent = (state = {}, action) => {
    switch (action.type) {
    case ActionTypes.REGISTER_POST_TYPE_COMPONENT_ID:
        if (action.data) {
            const nextState = {...state};
            if (action.data.postTypeComponentId == null) {
                return state;
            }
            nextState.id = action.data.postTypeComponentId;
            return nextState;
        }
        return state;
    default:
        return state;
    }
};
