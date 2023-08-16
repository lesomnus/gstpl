FROM ghcr.io/lesomnus/dev-golang:1.21

RUN apt-get update \
	&& apt-get install --yes --no-install-recommends \
		libgstreamer1.0-dev \
		gstreamer1.0-plugins-base \
	&& apt-get clean --yes \
	&& rm -rf /var/lib/apt/lists/*



USER hypnos

ENV GOPATH=/home/hypnos/go

# Comment out the following lines if you don't want to use Zsh.
RUN wget https://github.com/robbyrussell/oh-my-zsh/raw/master/tools/install.sh -O - | zsh || true \
	&& git clone https://github.com/zsh-users/zsh-autosuggestions         ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-autosuggestions     \
	&& git clone https://github.com/zsh-users/zsh-syntax-highlighting.git ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-syntax-highlighting \
	&& git clone https://github.com/zsh-users/zsh-completions             ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-completions         \
	&& git clone https://github.com/supercrabtree/k                       ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/k
ADD .zshrc /home/hypnos
ENV SHELL /bin/zsh
