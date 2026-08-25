import {connect} from 'react-redux';
import {bindActionCreators, type Dispatch} from 'redux';

import {getConfig} from 'mattermost-redux/selectors/entities/general';

import type {GlobalState} from '@mattermost/types/store';

import {fetchPollMetadata} from '@/actions/poll_metadata';
import {pollMetadata} from '@/selector';

import ActionView from '@/components/post_type/action_view/action_view';

function mapStateToProps(state: GlobalState) {
    const config = getConfig(state);
    return {
        siteUrl: config.SiteURL ?? '',
        pollMetadata: pollMetadata(state),
    };
}

function mapDispatchToProps(dispatch: Dispatch) {
    return {
        actions: bindActionCreators({
            fetchPollMetadata,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(ActionView);
