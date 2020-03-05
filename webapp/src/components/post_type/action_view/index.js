import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {fetchPollMetadata} from 'actions/poll_metadata';
import {pollMetadata} from 'selector';

import ActionView from './action_view';

function mapStateToProps(state) {
    const config = getConfig(state);
    return {
        siteUrl: config.SiteURL,
        pollMetadata: pollMetadata(state),
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            fetchPollMetadata,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(ActionView);
