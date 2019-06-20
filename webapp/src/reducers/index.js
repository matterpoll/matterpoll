import {combineReducers} from 'redux';

import {postTypeComponent} from './post_type';
import {votedAnswers} from './vote';

export default combineReducers({
    postTypeComponent,
    votedAnswers,
});