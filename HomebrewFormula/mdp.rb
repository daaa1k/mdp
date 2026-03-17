class Mdp < Formula
  desc "Paste clipboard image as Markdown link"
  homepage "https://github.com/daaa1k/mdp"
  version "0.2.3"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-macos-aarch64"
      sha256 "8fb87e1251a5b59bd6d37dd82be8121d36a91c94f78c9a4b3c999e24fba8c357" # macos-aarch64

      def install
        bin.install "mdp-macos-aarch64" => "mdp"
      end
    else
      url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-macos-x86_64"
      sha256 "" # macos-x86_64

      def install
        bin.install "mdp-macos-x86_64" => "mdp"
      end
    end
  end

  on_linux do
    url "https://github.com/daaa1k/mdp/releases/download/v#{version}/mdp-linux-x86_64"
    sha256 "87a7359718512eb72d9688895a23b10905efdcf56320ec6aa544b849908e109b" # linux-x86_64

    def install
      bin.install "mdp-linux-x86_64" => "mdp"
    end
  end

  test do
    system "#{bin}/mdp", "--help"
  end
end
