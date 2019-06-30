import React from 'react';
import PropTypes from 'prop-types';

import ActionButton from './action_button';

export default class ActionView extends React.PureComponent {
    static propTypes = {
        post: PropTypes.object.isRequired,
        attachment: PropTypes.object.isRequired,
        votedAnswers: PropTypes.object.isRequired,
        siteUrl: PropTypes.string.isRequired,

        actions: PropTypes.shape({
            fetchVotedAnswers: PropTypes.func.isRequired,
        }).isRequired,
    }

    componentDidMount() {
        this.props.actions.fetchVotedAnswers(this.props.siteUrl, this.props.post.props.poll_id);
    }

    render() {
        const actions = this.props.attachment.actions;
        if (!actions || !actions.length) {
            return '';
        }

        const content = [];
        const votedAnswers = this.props.votedAnswers || {};
        const answers = votedAnswers[this.props.post.props.poll_id] || {};

        actions.
            filter((action) => action.id && action.name).
            forEach((action) => {
                switch (action.type) {
                case 'button':
                    content.push(
                        <ActionButton
                            key={action.id}
                            action={action}
                            postId={this.props.post.id}
                            hasVoted={answers.voted_answers && (answers.voted_answers.indexOf(action.name) >= 0)}
                        />
                    );
                    break;
                default:
                    break;
                }
            });

        return (
            <div
                className='attachment-actions'
            >
                {content}
            </div>
        );
    }
}