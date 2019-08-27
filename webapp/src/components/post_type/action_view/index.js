import {connect} from 'react-redux';
import {bindActionCreators} from 'redux';

import {getConfig} from 'mattermost-redux/selectors/entities/general';

import {fetchVotedAnswers} from 'actions/vote';
import {votedAnswers} from 'selector';

import ActionView from './action_view';

function mapStateToProps(state) {
    const config = getConfig(state);
    return {
        siteUrl: config.SiteURL,
        votedAnswers: votedAnswers(state),
    };
}

function mapDispatchToProps(dispatch) {
    return {
        actions: bindActionCreators({
            fetchVotedAnswers,
        }, dispatch),
    };
}

export default connect(mapStateToProps, mapDispatchToProps)(ActionView);
