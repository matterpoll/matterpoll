import {combineReducers} from 'redux';

import {postTypeComponent} from './post_type';
import {pollMetadata} from './poll_metadata';

export default combineReducers({
    postTypeComponent,
    pollMetadata,
});
